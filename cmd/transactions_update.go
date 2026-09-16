package cmd

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newTransactionsUpdate(a *app) *cobra.Command {
	var output outputFlags
	var fields transactionFields
	var dryRun bool
	command := &cobra.Command{
		Use: "update ID... [field flags]", Short: "Change fields on one or more transactions",
		Long: `Apply the same field changes to every transaction named, through one
bulk update, and print the stored records in the shape of 'transactions
list'. At least one field flag is required; fields not named keep their
values. The flags are those of 'transactions create': --account, --date,
--amount, --payee or --transfer-to, --category, --memo (an empty value
clears it), --cleared, --approved (or --approved=false), and --flag (none
removes the flag).

There is no --split, because the API cannot change the lines of an
existing split or turn a transaction into one, and no --import-id,
because the API treats it as a way to name a transaction rather than a
field to set. On a split, the API ignores --date, --amount, and
--category. --transfer-to with --category is refused when the transaction
and the target are both plan accounts.

Requests: the plans endpoint, the accounts endpoint, the payees endpoint
with --payee, the categories endpoint with --category, then one bulk
update per 100 transactions. --transfer-to with --category and without
--account reads the current records first, as a dry run does, to find
each transaction's account.

` + writeHelp + "\n\n" + bulkHelp + "\n\n" + planHelp + "\n\n" + dateFlagHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions update 7a3e... --memo "reimbursed" --flag green --dry-run
  ynab transactions update 7a3e... 9c1d... --category "Bills: Internet"
  ynab transactions update 7a3e... --cleared reconciled --allow-writes`,
		Args: cobra.MinimumNArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			if !fields.given(cmd) {
				return usageError("give at least one field flag to change")
			}

			s, err := a.openWrite(cmd, dryRun)
			if err != nil {
				return err
			}
			change, transfer, err := fields.resolve(cmd, s)
			if err != nil {
				return err
			}
			write := bulkWrite{base: "update", change: change, ids: args, past: "updated"}
			if change.PayeeName != nil {
				write.note = fmt.Sprintf("and created the payee %q", *change.PayeeName)
			}
			if transfer != nil && change.CategoryID != nil {
				if change.AccountID != "" {
					if err := transferCategoryConflict(s.accountByID(change.AccountID), *transfer); err != nil {
						return err
					}
				} else {
					write.check = func(tx ynab.Transaction) error {
						return transferCategoryConflict(s.accountByID(tx.AccountID), *transfer)
					}
				}
			}
			return s.bulk(cmd, output, write)
		}),
	}
	output.bind(command, true)
	fields.bind(command)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Preview the records without changing them")
	return command
}
