package cli

import (
	"net/http"

	"github.com/spf13/cobra"
)

func newAPIPut(a *app) *cobra.Command {
	var options apiOptions
	command := &cobra.Command{
		Use: "put PATH", Short: "Send a PUT request and print the raw response",
		Long: `Send a PUT request to PATH, relative to /v1 with {plan}
replaced by the configured plan's ID, and print the response body as
returned. See 'ynab api --help' for the contract: raw milliunits and API
field names, --pretty, and the exit status 1 with the body on stdout and
the status on stderr for a non-2xx response.`,
		Example: "  ynab api put " + "/plans/{plan}/scheduled_transactions/ID --body-file st.json \\\n    --allow-writes",
	}
	command.Flags().BoolVar(&options.pretty, "pretty", false, "Re-indent the JSON response")
	command.Flags().StringVar(&options.body, "body", "", "JSON request body")
	command.Flags().StringVar(&options.bodyFile, "body-file", "", "File holding the JSON request body; - reads stdin")
	command.MarkFlagsMutuallyExclusive("body", "body-file")
	command.Flags().BoolVar(&options.dryRun, "dry-run", false, "Print the request that would be sent without sending it")
	return newAPIVerb(a, command, http.MethodPut, true, &options)
}
