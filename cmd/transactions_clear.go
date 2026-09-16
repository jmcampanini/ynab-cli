package cmd

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newTransactionsClear(a *app) *cobra.Command {
	return newClearedStateCommand(a, "clear", "cleared", "Mark transactions cleared")
}

// newClearedStateCommand builds clear and unclear, which differ only in
// the state they set; both refuse to touch a reconciled transaction
// without --force.
func newClearedStateCommand(a *app, verb, state, short string) *cobra.Command {
	var output outputFlags
	var dryRun, force bool
	command := &cobra.Command{
		Use: verb + " ID...", Short: short,
		Long: fmt.Sprintf(`Set the cleared state of the transactions named to %s through one bulk
update and print the stored records in the shape of 'transactions list'.
A reconciled transaction is refused unless --force is given, because
changing its state undoes the reconciliation in the app; the check reads
the current records first. There is no account-level reconcile in the
API; 'transactions update --cleared reconciled' marks one transaction.

Requests: the plans endpoint, the accounts endpoint, the plan listing
that checks the current state, one request per ID older than a year,
then one bulk update per 100 transactions.

`, state) + writeHelp + "\n\n" + bulkHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: fmt.Sprintf(`  ynab transactions %s 7a3e... 9c1d... --dry-run
  ynab transactions %s 7a3e... --force --allow-writes`, verb, verb),
		Args: cobra.MinimumNArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			s, err := a.openWrite(cmd, dryRun)
			if err != nil {
				return err
			}

			check := func(tx ynab.Transaction) error {
				if tx.Cleared == "reconciled" && !force {
					return fmt.Errorf("transaction %s is reconciled; --force marks it %s, which undoes the reconciliation in the app", tx.ID, state)
				}
				return nil
			}
			return s.bulk(cmd, output, bulkWrite{base: verb, change: ynab.SaveTransaction{Cleared: state}, check: check, ids: args, past: state})
		}),
	}
	output.bind(command, true)
	command.Flags().BoolVar(&force, "force", false, "Change a reconciled transaction too")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Preview the records without changing them")
	return command
}
