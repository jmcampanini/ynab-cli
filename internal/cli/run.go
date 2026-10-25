package cli

import (
	"github.com/jmcampanini/ynab-cli/internal/status"
	"github.com/spf13/cobra"
)

// run wraps a leaf runner so its failures exit ExitFailure unless they
// already carry a status.
func run(runner func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		return status.CommandFailure(runner(cmd, args))
	}
}
