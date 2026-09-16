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
category since the plan's first month. Tracking-account transactions
and transfers between plan accounts are skipped because they do not
need categories; their count goes to stderr. --category takes an
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
				write.current = s.transactionsToCategorize(cmd, transactions)
			}
			return s.bulk(cmd, output, write)
		}),
	}
	output.bind(command, true)
	command.Flags().StringVar(&category, "category", "", "Category ID, exact name, or \"Group: Name\"")
	command.Flags().BoolVar(&allUncategorized, "all-uncategorized", false, "Categorize every transaction without a category")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be categorized without changing it")
	_ = command.MarkFlagRequired("category")
	completeFlags(command, a.completeCategories(), "category")
	return command
}

// transactionsToCategorize reports candidates that do not need categories.
// The result is never nil, so an empty selection does not trigger a bulk
// write's lookup by ID.
func (s *writeSession) transactionsToCategorize(cmd *cobra.Command, transactions []ynab.Transaction) []ynab.Transaction {
	kept := transactionsNeedingCategory(transactions, s.accounts)
	skipped := len(transactions) - len(kept)
	if skipped > 0 {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "skipped %s: these transactions do not need a category\n", transactionCount(skipped))
	}
	if kept == nil {
		return []ynab.Transaction{}
	}
	return kept
}
