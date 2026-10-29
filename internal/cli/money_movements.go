package cli

import (
	"github.com/spf13/cobra"
)

func newMoneyMovements(a *app) *cobra.Command {
	command := &cobra.Command{
		Use: "money-movements", Short: "List the plan's recorded money movements",
		Long: `Read the money movements of the configured plan: each recorded move of
assigned money between two categories, or between a category and ready
to assign. A bare 'ynab money-movements' prints this help and exits 0.
The API records movements but cannot create them; 'ynab months move'
makes a move as two assigned-amount writes, which the API may record.

` + planHelp + "\n\n" + monthHelp,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error { return cmd.Help() }),
	}
	command.AddCommand(newMoneyMovementsList(a))
	return command
}
