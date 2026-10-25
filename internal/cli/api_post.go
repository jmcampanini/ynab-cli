package cli

import (
	"net/http"

	"github.com/spf13/cobra"
)

func newAPIPost(a *app) *cobra.Command {
	var options apiOptions
	command := &cobra.Command{
		Use: "post PATH", Short: "Send a POST request and print the raw response",
		Long: `Send a POST request to PATH, relative to /v1 with {plan}
replaced by the configured plan's ID, and print the response body as
returned. See 'ynab api --help' for the contract: raw milliunits and API
field names, --pretty, and the exit status 1 with the body on stdout and
the status on stderr for a non-2xx response.`,
		Example: "  ynab api post " + "/plans/{plan}/payees --body '{\"payee\":{\"name\":\"Bakery\"}}' \\\n    --allow-writes",
	}
	command.Flags().BoolVar(&options.pretty, "pretty", false, "Re-indent the JSON response")
	command.Flags().StringVar(&options.body, "body", "", "JSON request body")
	command.Flags().StringVar(&options.bodyFile, "body-file", "", "File holding the JSON request body; - reads stdin")
	command.MarkFlagsMutuallyExclusive("body", "body-file")
	command.Flags().BoolVar(&options.dryRun, "dry-run", false, "Print the request that would be sent without sending it")
	return newAPIVerb(a, command, http.MethodPost, true, &options)
}
