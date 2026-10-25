package cli

import (
	"github.com/spf13/cobra"
)

func newTransactionsUnclear(a *app) *cobra.Command {
	var options clearedStateOptions
	command := &cobra.Command{
		Use: "unclear ID...", Short: "Mark transactions uncleared",
		Long: `Set the cleared state of the transactions named to uncleared through one bulk
update and print the stored records in the shape of 'transactions list'.
A reconciled transaction is refused unless --force is given, because
changing its state undoes the reconciliation in the app; the check reads
the current records first. There is no account-level reconcile in the
API; 'transactions update --cleared reconciled' marks one transaction.

Requests: the plans endpoint, the accounts endpoint, the plan listing
that checks the current state, one request per ID older than a year,
then one bulk update per 100 transactions.

` + writeHelp + "\n\n" + bulkHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions unclear 7a3e... 9c1d... --dry-run
  ynab transactions unclear 7a3e... --force --allow-writes`,
		Args: cobra.MinimumNArgs(1),
		RunE: run(a.setClearedState("unclear", "uncleared", &options)),
	}
	options.bind(command)
	return command
}
