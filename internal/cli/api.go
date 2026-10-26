package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/selection"
	"github.com/jmcampanini/ynab-cli/internal/status"
	"github.com/spf13/cobra"
)

// planPlaceholder is the PATH token the configured plan's ID replaces.
const planPlaceholder = "{plan}"

// apiOptions are the flag values of one api verb. The write verbs add a
// body and --dry-run to --pretty.
type apiOptions struct {
	body, bodyFile string
	pretty, dryRun bool
}

func (o *apiOptions) bindRead(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.pretty, "pretty", false, "Re-indent the JSON response")
}

func (o *apiOptions) bindWrite(cmd *cobra.Command) {
	o.bindRead(cmd)
	cmd.Flags().StringVar(&o.body, "body", "", "JSON request body")
	cmd.Flags().StringVar(&o.bodyFile, "body-file", "", "File holding the JSON request body; - reads stdin")
	cmd.MarkFlagsMutuallyExclusive("body", "body-file")
	cmd.Flags().BoolVar(&o.dryRun, "dry-run", false, "Print the request without sending it")
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

Each verb makes one request, plus the plans endpoint when PATH names
{plan}. Every verb but get is a write and takes a request body. The
API's reference at https://api.ynab.com documents every path and body.

` + apiBodyHelp + "\n\n" + writeHelp + "\n\n" + planHelp + "\n\n" + configHelp,
		Example: `  ynab api get /user
  ynab api get "/plans/{plan}/transactions?since_date=2026-09-01" --pretty
  ynab api patch /plans/{plan}/months/current/categories/ID \
    --body '{"category":{"budgeted":1000}}' --dry-run
  ynab api post /plans/{plan}/payees --body-file payee.json --allow-writes
  ynab api delete /plans/{plan}/transactions/ID --allow-writes`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(newAPIGet(a), newAPIPost(a), newAPIPatch(a), newAPIPut(a), newAPIDelete(a))
	return command
}

// apiBodyHelp is the request body contract the api group and each write
// verb share, so the file a verb reads is named in its own help.
const apiBodyHelp = `Request body:
  --body gives the JSON directly; --body-file PATH reads it from that
  file, resolving a relative path against the working directory, and
  --body-file - reads stdin. The body must parse as JSON. --dry-run prints
  the method, URL, and body that would be sent without sending them.`

// newAPIVerb gives a verb command its operand rule and runner. writes
// marks the verbs that go through the write gate.
func newAPIVerb(a *app, command *cobra.Command, method string, writes bool, options *apiOptions) {
	command.Args = cobra.ExactArgs(1)
	command.RunE = run(func(cmd *cobra.Command, args []string) error {
		path := args[0]
		if path == "" {
			return status.Usage("PATH must not be empty")
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		payload, err := requestBody(cmd, options.body, options.bodyFile)
		if err != nil {
			return err
		}

		loaded, client, err := a.connect(cmd)
		if err != nil {
			return err
		}
		if writes && !options.dryRun && !loaded.Config.AllowWrites {
			return status.WritesDisabled(loaded.Path)
		}
		if strings.Contains(path, planPlaceholder) {
			plan, err := selection.Plan(cmd.Context(), client, loaded.Config)
			if err != nil {
				return err
			}
			path = strings.ReplaceAll(path, planPlaceholder, plan.ID)
		}

		out := cmd.OutOrStdout()
		if options.dryRun {
			if _, err := fmt.Fprintln(out, method+" "+client.URL(path)); err != nil {
				return err
			}
			return writeBody(out, payload, options.pretty)
		}
		response, err := client.Raw(cmd.Context(), method, path, payload)
		if err != nil {
			return err
		}
		if err := writeBody(out, response.Body, options.pretty); err != nil {
			return err
		}
		if response.Status < 200 || response.Status > 299 {
			return fmt.Errorf("HTTP %d %s", response.Status, http.StatusText(response.Status))
		}
		return nil
	})
}

// requestBody reads the body from --body or --body-file and checks that
// it is JSON before any request. Nil means no body.
func requestBody(cmd *cobra.Command, body, bodyFile string) ([]byte, error) {
	var payload []byte
	switch {
	case cmd.Flags().Changed("body"):
		payload = []byte(body)
	case cmd.Flags().Changed("body-file") && bodyFile == "":
		return nil, status.Usage("--body-file requires a file path or -")
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
		return nil, status.Usage("the request body is not valid JSON")
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
