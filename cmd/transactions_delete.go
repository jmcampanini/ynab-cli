package cmd

import (
	"github.com/spf13/cobra"
)

func newTransactionsDelete(a *app) *cobra.Command {
	var output outputFlags
	var dryRun bool
	command := &cobra.Command{
		Use: "delete ID", Short: "Delete one transaction and print it",
		Long: `Delete one transaction of the configured plan by its ID and print the
record as it was, in the shape of 'transactions get', so it can be
recreated with 'transactions create'. Deleting one side of a transfer
deletes both. There is no undo. Three requests: the plans endpoint, the
accounts endpoint, then the delete; a dry run reads the transaction
instead.

` + writeHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions delete 7a3e... --dry-run
  ynab transactions delete 7a3e... --allow-writes --jsonl >> deleted.jsonl`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			s, err := a.openTransactionWrite(cmd, dryRun)
			if err != nil {
				return err
			}

			fetch := s.client.DeleteTransaction
			if s.dryRun {
				fetch = s.client.Transaction
			}
			tx, err := fetch(cmd.Context(), s.plan.ID, args[0])
			if err != nil {
				return err
			}
			return s.printTransaction(cmd.OutOrStdout(), output, tx, s.summary("deleted", "delete", 1))
		}),
	}
	output.bind(command, false)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print the transaction without deleting it")
	return command
}
