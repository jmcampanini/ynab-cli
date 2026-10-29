package cli

import (
	"net/http"

	"github.com/spf13/cobra"
)

func newAPIDelete(a *app) *cobra.Command {
	var options apiOptions
	command := &cobra.Command{
		Use: "delete PATH", Short: "Send a DELETE request and print the raw response",
		Long: `Send a DELETE request to PATH, relative to /v1 with {plan}
replaced by the configured plan's ID, and print the response body as
returned. See 'ynab api --help' for the contract: raw milliunits and API
field names, and --pretty. A non-2xx response exits 1 with an HTTP status
and available API error detail on stderr; stdout stays empty.

` + apiBodyHelp,
		Example: "  ynab api delete /plans/{plan}/transactions/ID --allow-writes",
	}
	options.bindWrite(command)
	newAPIVerb(a, command, http.MethodDelete, true, &options)
	return command
}
