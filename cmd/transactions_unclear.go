package cmd

import "github.com/spf13/cobra"

func newTransactionsUnclear(a *app) *cobra.Command {
	return newClearedStateCommand(a, "unclear", "uncleared", "Mark transactions uncleared")
}
