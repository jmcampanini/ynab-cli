package cli

import (
	"errors"
	"fmt"

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
deletes both. There is no undo. An ordinary delete makes four requests:
plans, accounts, the transaction, then DELETE. A dry run makes only the
three reads and does not create temporary entries.

Deleting a split can leave YNAB's category totals counting the deleted lines.
After deleting a split in an on-plan account, create and delete one temporary
zero-amount entry per distinct category on its live, non-transfer lines.
Each entry uses the original account and date. Uncategorized lines and
transfers are excluded. This adds two writes per category, without reading
or comparing category totals. Ordinary transactions get no temporary entries.

Completed create/delete pairs are reported on stderr with code
split-delete-recalculated in every output mode. This confirms the workaround
ran; category totals are not independently verified. If a temporary-entry
operation fails, print the deleted record and exit 1, explaining that the
original deletion already happened and later categories were not processed.
A failed temporary-entry deletion names its ID for manual cleanup. A create
with an uncertain result names its memo; inspect the register before retrying.
The raw 'api delete' command sends only the requested DELETE and does not
check or repair totals.

` + writeHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions delete 7a3e... --dry-run
  ynab transactions delete 7a3e... --allow-writes --jsonl >> deleted.jsonl`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			s, err := a.openTransactionWrite(cmd, dryRun)
			if err != nil {
				return err
			}

			tx, err := s.client.Transaction(cmd.Context(), s.plan.ID, args[0])
			if err != nil {
				return err
			}
			if s.dryRun {
				return s.printTransaction(cmd.OutOrStdout(), output, tx, s.summary("deleted", "delete", 1))
			}
			deleted, err := s.client.DeleteTransaction(cmd.Context(), s.plan.ID, args[0])
			if err != nil {
				return err
			}

			repairErr := s.repairSplitDeletion(cmd.Context(), tx, cmd.ErrOrStderr())
			if repairErr != nil {
				repairErr = fmt.Errorf("transaction %s was deleted, but category recalculation stopped; later categories were not processed: %w", tx.ID, repairErr)
			}
			printErr := s.printTransaction(cmd.OutOrStdout(), output, deleted, s.summary("deleted", "delete", 1))
			return errors.Join(repairErr, printErr)
		}),
	}
	output.bind(command, false)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print the transaction without deleting it")
	return command
}
