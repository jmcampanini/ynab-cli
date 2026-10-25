package cli

import "github.com/spf13/cobra"

func newTransactionsUnclear(a *app) *cobra.Command {
	var options clearedStateOptions
	command := newClearedStateCommand(a, "unclear", "uncleared", "Mark transactions uncleared", &options)
	options.output.bind(command, true)
	command.Flags().BoolVar(&options.force, "force", false, "Change a reconciled transaction too")
	command.Flags().BoolVar(&options.dryRun, "dry-run", false, "Preview the records without changing them")
	return command
}
