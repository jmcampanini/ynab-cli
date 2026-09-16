package cmd

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newTransactionsCategorize(a *app) *cobra.Command {
	var output outputFlags
	var dryRun, allUncategorized bool
	var category string
	command := &cobra.Command{
		Use: "categorize --category C ID... | --all-uncategorized", Short: "Set the category of transactions",
		Long: `Set one category on transactions through one bulk update and print the
stored records in the shape of 'transactions list'. Name them by ID, or
pass --all-uncategorized to categorize every transaction without a
category since the plan's first month. The API lists transfers between
plan accounts as uncategorized although they carry no category, so
those are skipped and their count goes to stderr. --category takes an
ID, an exact name, or "Group: Name"; a Credit Card Payments category is
refused, since the API ignores it. The API also ignores a category on a
split; a real run says so on stderr.

Requests: the plans endpoint, the accounts endpoint, the categories
endpoint, the uncategorized listing with --all-uncategorized, then one
bulk update per 100 transactions.

` + writeHelp + "\n\n" + bulkHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions categorize --category Groceries 7a3e... 9c1d... --dry-run
  ynab transactions categorize --all-uncategorized --category "Fun: Misc"`,
		Args: cobra.ArbitraryArgs,
		RunE: run(func(cmd *cobra.Command, args []string) error {
			if err := oneOfIDsOrAll(args, "all-uncategorized", allUncategorized); err != nil {
				return err
			}

			s, err := a.openTransactionWrite(cmd, dryRun)
			if err != nil {
				return err
			}
			found, err := s.category(cmd.Context(), category)
			if err != nil {
				return err
			}
			write := bulkWrite{base: "categorize", change: ynab.SaveTransaction{CategoryID: &found.ID}, ids: args, past: "categorized"}
			if allUncategorized {
				transactions, err := s.client.Transactions(cmd.Context(), s.plan.ID, ynab.TransactionFilter{SinceDate: s.plan.FirstMonth, Type: ynab.TypeUncategorized})
				if err != nil {
					return err
				}
				write.current = s.withoutPlanTransfers(cmd, transactions)
			}
			return s.bulk(cmd, output, write)
		}),
	}
	output.bind(command, true)
	command.Flags().StringVar(&category, "category", "", "Category ID, exact name, or \"Group: Name\"")
	command.Flags().BoolVar(&allUncategorized, "all-uncategorized", false, "Categorize every transaction without a category")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be categorized without changing it")
	_ = command.MarkFlagRequired("category")
	return command
}

// withoutPlanTransfers drops transfers between two plan accounts, which
// the API lists as uncategorized but gives no category, and reports the
// count on stderr. The result is never nil.
func (s *writeSession) withoutPlanTransfers(cmd *cobra.Command, transactions []ynab.Transaction) []ynab.Transaction {
	kept := []ynab.Transaction{}
	skipped := 0
	for _, tx := range transactions {
		if tx.TransferAccountID != nil && s.accountByID(tx.AccountID).OnPlan && s.accountByID(*tx.TransferAccountID).OnPlan {
			skipped++
			continue
		}
		kept = append(kept, tx)
	}
	if skipped > 0 {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "skipped %s: transfers between plan accounts carry no category\n", transactionCount(skipped))
	}
	return kept
}
