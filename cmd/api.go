package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// planPlaceholder is the PATH token the configured plan's ID replaces.
const planPlaceholder = "{plan}"

// apiVerb is one HTTP method exposed as an 'api' subcommand.
type apiVerb struct {
	method string
	// writes marks the verbs that go through the write gate and take a
	// body and --dry-run.
	writes bool
}

var apiVerbs = []apiVerb{
	{method: http.MethodGet},
	{method: http.MethodPost, writes: true},
	{method: http.MethodPatch, writes: true},
	{method: http.MethodPut, writes: true},
	{method: http.MethodDelete, writes: true},
}

func newAPI(a *app) *cobra.Command {
	command := &cobra.Command{
		Use: "api", Short: "Send a raw request to the YNAB API",
		Long: `Send one request to the YNAB API as given and print the response body
as returned. This is the escape hatch for anything the other commands do
not cover: the output shape is YNAB's, not this CLI's, so amounts are
milliunits (1000 per currency unit) and field names are the API's, such
as budgeted, balance, and goal_target. Nothing else in ynab prints
milliunits. A bare 'ynab api' prints this help and exits 0.

PATH is relative to https://api.ynab.com/v1 and may carry a query
string; the token {plan} in it becomes the configured plan's ID. The
body of a 2xx response goes to stdout as returned, or re-indented with
--pretty. For any other status the body still goes to stdout, the
status goes to stderr, and the exit status is 1, so the API's own error
detail is never lost. Requests count against the API's limit of 200 per
hour per token like any other.

'get' makes one request, plus the plans endpoint when PATH names
{plan}. 'post', 'patch', 'put', and 'delete' are writes: they take the
JSON body from --body or --body-file (- for stdin), which must parse as
JSON, and --dry-run prints the method, URL, and body that would be sent
without sending them. The API's reference at https://api.ynab.com
documents every path and body.

` + writeHelp + "\n\n" + planHelp + "\n\n" + configHelp,
		Example: `  ynab api get /user
  ynab api get "/plans/{plan}/transactions?since_date=2026-09-01" --pretty
  ynab api patch /plans/{plan}/months/current/categories/ID \
    --body '{"category":{"budgeted":1000}}' --dry-run
  ynab api post /plans/{plan}/payees --body-file payee.json --allow-writes
  ynab api delete /plans/{plan}/transactions/ID --allow-writes`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	for _, verb := range apiVerbs {
		command.AddCommand(newAPIVerb(a, verb))
	}
	return command
}

func newAPIVerb(a *app, verb apiVerb) *cobra.Command {
	var body, bodyFile string
	var pretty, dryRun bool
	name := strings.ToLower(verb.method)
	command := &cobra.Command{
		Use: name + " PATH", Short: "Send a " + verb.method + " request and print the raw response",
		Long: `Send a ` + verb.method + ` request to PATH, relative to /v1 with {plan}
replaced by the configured plan's ID, and print the response body as
returned. See 'ynab api --help' for the contract: raw milliunits and API
field names, --pretty, and the exit status 1 with the body on stdout and
the status on stderr for a non-2xx response.`,
		Example: "  ynab api " + name + " " + verbExamples[verb.method],
		Args:    cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			path := args[0]
			if path == "" {
				return usageError("PATH must not be empty")
			}
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			payload, err := requestBody(cmd, body, bodyFile)
			if err != nil {
				return err
			}

			loaded, client, err := a.connect(cmd)
			if err != nil {
				return err
			}
			if verb.writes && !dryRun && !loaded.Config.AllowWrites {
				return writesDisabled(loaded.Path)
			}
			if strings.Contains(path, planPlaceholder) {
				plan, err := selectedPlan(cmd.Context(), client, loaded.Config)
				if err != nil {
					return err
				}
				path = strings.ReplaceAll(path, planPlaceholder, plan.ID)
			}

			out := cmd.OutOrStdout()
			if dryRun {
				if _, err := fmt.Fprintln(out, verb.method+" "+client.URL(path)); err != nil {
					return err
				}
				return writeBody(out, payload, pretty)
			}
			response, err := client.Raw(cmd.Context(), verb.method, path, payload)
			if err != nil {
				return err
			}
			if err := writeBody(out, response.Body, pretty); err != nil {
				return err
			}
			if response.Status < 200 || response.Status > 299 {
				return fmt.Errorf("HTTP %d %s", response.Status, http.StatusText(response.Status))
			}
			return nil
		}),
	}
	command.Flags().BoolVar(&pretty, "pretty", false, "Re-indent the JSON response")
	if verb.writes {
		command.Flags().StringVar(&body, "body", "", "JSON request body")
		command.Flags().StringVar(&bodyFile, "body-file", "", "File holding the JSON request body; - reads stdin")
		command.MarkFlagsMutuallyExclusive("body", "body-file")
		command.Flags().BoolVar(&dryRun, "dry-run", false, "Print the request that would be sent without sending it")
	}
	return command
}

var verbExamples = map[string]string{
	http.MethodGet:    "/user",
	http.MethodPost:   "/plans/{plan}/payees --body '{\"payee\":{\"name\":\"Bakery\"}}' \\\n    --allow-writes",
	http.MethodPatch:  "/plans/{plan}/categories/ID \\\n    --body '{\"category\":{\"note\":\"x\"}}' --dry-run",
	http.MethodPut:    "/plans/{plan}/scheduled_transactions/ID --body-file st.json \\\n    --allow-writes",
	http.MethodDelete: "/plans/{plan}/transactions/ID --allow-writes",
}

// requestBody reads the body from --body or --body-file and checks that
// it is JSON before any request. Nil means no body.
func requestBody(cmd *cobra.Command, body, bodyFile string) ([]byte, error) {
	var payload []byte
	switch {
	case cmd.Flags().Changed("body"):
		payload = []byte(body)
	case cmd.Flags().Changed("body-file") && bodyFile == "":
		return nil, usageError("--body-file requires a file path or -")
	case bodyFile == "-":
		read, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, fmt.Errorf("read body from stdin: %w", err)
		}
		payload = read
	case bodyFile != "":
		read, err := os.ReadFile(bodyFile)
		if err != nil {
			return nil, fmt.Errorf("read body file: %w", err)
		}
		payload = read
	default:
		return nil, nil
	}
	if !json.Valid(payload) {
		return nil, usageError("the request body is not valid JSON")
	}
	return payload, nil
}

// writeBody prints a body as given, or re-indented when pretty and it
// is JSON, ending it with a newline when it lacks one. An empty body
// prints nothing.
func writeBody(w io.Writer, body []byte, pretty bool) error {
	if len(body) == 0 {
		return nil
	}
	if pretty && json.Valid(body) {
		var indented bytes.Buffer
		if err := json.Indent(&indented, body, "", "  "); err != nil {
			return err
		}
		body = indented.Bytes()
	}
	if !bytes.HasSuffix(body, []byte("\n")) {
		body = append(body, '\n')
	}
	_, err := w.Write(body)
	return err
}
