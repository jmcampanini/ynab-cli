package cmd

import (
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newTransactionsApprove(a *app) *cobra.Command {
	var output outputFlags
	var dryRun, allUnapproved bool
	var account string
	command := &cobra.Command{
		Use: "approve ID... | --all-unapproved [--account A]", Short: "Approve transactions",
		Long: `Mark transactions approved through one bulk update and print the stored
records in the shape of 'transactions list'. Name them by ID, or pass
--all-unapproved to approve every transaction awaiting approval since
the plan's first month, within one account with --account (an ID or
exact name). --all-unapproved lists the unapproved transactions first;
with --dry-run that list, marked approved, is the whole output.

Requests: the plans endpoint, the accounts endpoint, the unapproved
listing with --all-unapproved, then one bulk update per 100 transactions.

` + writeHelp + "\n\n" + bulkHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions approve --all-unapproved --dry-run
  ynab transactions approve --all-unapproved --account Visa --allow-writes
  ynab transactions review --jsonl | jq -r .id \
    | xargs ynab transactions approve --allow-writes`,
		Args: cobra.ArbitraryArgs,
		RunE: run(func(cmd *cobra.Command, args []string) error {
			if err := oneOfIDsOrAll(args, "all-unapproved", allUnapproved); err != nil {
				return err
			}
			if account != "" && !allUnapproved {
				return usageError("--account narrows --all-unapproved; with IDs, name the transactions alone")
			}

			s, err := a.openWrite(cmd, dryRun)
			if err != nil {
				return err
			}
			approved := true
			write := bulkWrite{base: "approve", change: ynab.SaveTransaction{Approved: &approved}, ids: args, past: "approved"}
			if allUnapproved {
				if write.current, err = s.listUnapproved(cmd, account); err != nil {
					return err
				}
			}
			return s.bulk(cmd, output, write)
		}),
	}
	output.bind(command, true)
	command.Flags().BoolVar(&allUnapproved, "all-unapproved", false, "Approve every transaction awaiting approval")
	command.Flags().StringVar(&account, "account", "", "With --all-unapproved, only this account's transactions")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be approved without approving")
	return command
}

// listUnapproved fetches the unapproved transactions of the plan, or of
// one account, back to the plan's first month rather than the API's
// one-year default. The result is never nil, so an empty list still
// counts as a selection.
func (s *writeSession) listUnapproved(cmd *cobra.Command, account string) ([]ynab.Transaction, error) {
	filter := ynab.TransactionFilter{SinceDate: s.plan.FirstMonth, Type: ynab.TypeUnapproved}
	var transactions []ynab.Transaction
	var err error
	if account == "" {
		transactions, err = s.client.Transactions(cmd.Context(), s.plan.ID, filter)
	} else {
		found, findErr := s.account(account)
		if findErr != nil {
			return nil, findErr
		}
		transactions, err = s.client.AccountTransactions(cmd.Context(), s.plan.ID, found.ID, filter)
	}
	if err != nil {
		return nil, err
	}
	if transactions == nil {
		transactions = []ynab.Transaction{}
	}
	return transactions, nil
}
