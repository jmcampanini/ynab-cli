package cli

import (
	"net/http"

	"github.com/spf13/cobra"
)

func newAPIGet(a *app) *cobra.Command {
	var options apiOptions
	command := &cobra.Command{
		Use: "get PATH", Short: "Send a GET request and print the raw response",
		Long: `Send a GET request to PATH, relative to /v1 with {plan}
replaced by the configured plan's ID, and print the response body as
returned. See 'ynab api --help' for the contract: raw milliunits and API
field names, --pretty, and the exit status 1 with the body on stdout and
the status on stderr for a non-2xx response.`,
		Example: "  ynab api get " + "/user",
	}
	command.Flags().BoolVar(&options.pretty, "pretty", false, "Re-indent the JSON response")
	return newAPIVerb(a, command, http.MethodGet, false, &options)
}
