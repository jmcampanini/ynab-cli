package cli

import (
	"net/http"

	"github.com/spf13/cobra"
)

func newAPIPatch(a *app) *cobra.Command {
	var options apiOptions
	command := &cobra.Command{
		Use: "patch PATH", Short: "Send a PATCH request and print the raw response",
		Long: `Send a PATCH request to PATH, relative to /v1 with {plan}
replaced by the configured plan's ID, and print the response body as
returned. See 'ynab api --help' for the contract: raw milliunits and API
field names, and --pretty. A non-2xx response exits 1 with an HTTP status
and available API error detail on stderr; stdout stays empty.

` + apiBodyHelp,
		Example: "  ynab api patch /plans/{plan}/categories/ID \\\n    --body '{\"category\":{\"note\":\"x\"}}' --dry-run",
	}
	options.bindWrite(command)
	newAPIVerb(a, command, http.MethodPatch, true, &options)
	return command
}
