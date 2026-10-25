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
		Example: "  ynab api put /plans/{plan}/scheduled_transactions/ID --body-file st.json \\\n    --allow-writes",
	}
	options.bindWrite(command)
	newAPIVerb(a, command, http.MethodPut, true, &options)
	return command
}
