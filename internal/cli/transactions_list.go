package cli

import (
	"fmt"
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/dates"
	"github.com/jmcampanini/ynab-cli/internal/register"
	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/selection"
	"github.com/jmcampanini/ynab-cli/internal/status"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// transactionFilters is every list filter after name resolution and date
// parsing. Empty strings and nil amounts mean the filter is off.
type transactionFilters struct {
	register.Query
	categoryID string
	cleared    string
	flag       string
	max        *ynab.Amount
	memo       string
	min        *ynab.Amount
	payeeID    string
}

// byName reports whether a category or payee filter is on, which is when
// the summary totals matching lines rather than whole transactions.
func (f transactionFilters) byName() bool {
	return f.categoryID != "" || f.payeeID != ""
}

// matching applies the client-side filters and returns the amount the
// summary counts: the transaction's amount, or under a category or payee
// filter the total of the lines that match. A line without a payee
// inherits the parent's. A split whose own payee matches while every line
// names another payee is kept with its whole amount.
func (f transactionFilters) matching(tx ynab.Transaction) (ynab.Amount, bool) {
	if f.Unapproved && tx.Approved ||
		f.cleared != "" && tx.Cleared != f.cleared ||
		f.flag != "" && !f.flagMatches(tx) ||
		f.min != nil && tx.Amount < *f.min ||
		f.max != nil && tx.Amount > *f.max ||
		f.memo != "" && !memoMatches(tx, f.memo) {
		return 0, false
	}
	if !f.byName() {
		return tx.Amount, true
	}
	if len(tx.Subtransactions) == 0 {
		if f.nameMatches(tx.CategoryID, tx.PayeeID) {
			return tx.Amount, true
		}
		return 0, false
	}
	var total ynab.Amount
	matched := false
	for _, line := range tx.Subtransactions {
		payeeID := line.PayeeID
		if payeeID == nil {
			payeeID = tx.PayeeID
		}
		if f.nameMatches(line.CategoryID, payeeID) {
			total += line.Amount
			matched = true
		}
	}
	if !matched && f.nameMatches(tx.CategoryID, tx.PayeeID) {
		return tx.Amount, true
	}
	return total, matched
}

func (f transactionFilters) nameMatches(categoryID, payeeID *string) bool {
	return (f.categoryID == "" || categoryID != nil && *categoryID == f.categoryID) &&
		(f.payeeID == "" || payeeID != nil && *payeeID == f.payeeID)
}

func (f transactionFilters) flagMatches(tx ynab.Transaction) bool {
	color := render.Text(tx.FlagColor)
	if f.flag == "none" {
		return color == ""
	}
	return color == f.flag
}

func memoMatches(tx ynab.Transaction, memo string) bool {
	if containsFold(render.Text(tx.Memo), memo) {
		return true
	}
	for _, line := range tx.Subtransactions {
		if containsFold(render.Text(line.Memo), memo) {
			return true
		}
	}
	return false
}

func newTransactionsList(a *app) *cobra.Command {
	var output outputFlags
	var account, category, payee, since, until, month, cleared, flag, memo, minText, maxText string
	var unapproved, uncategorized bool
	command := &cobra.Command{
		Use: "list", Short: "List the plan's transactions with filters",
		Long: `List the configured plan's transactions, oldest first, with one row per
transaction and a split's lines indented beneath it under the category
"Split". Columns: date, account, payee, category, memo, amount, cleared,
approved, and flag. The summary line carries the count and the net total.

Filters combine with AND. --account, --category, and --payee take an ID
or the exact name, matched case-insensitively; categories also take
"Group: Name". --category and --payee match a transaction whose own
category or payee matches, or any line of a split that does; the whole
transaction is shown and the summary totals only the matching lines.
--since and --until bound the date range; --month is that month's range.
Without --since or --month the API returns one year back. --unapproved
and --uncategorized keep transactions awaiting approval or a category.
--uncategorized excludes tracking-account transactions and transfers
between plan accounts, since those do not need categories.
--cleared takes uncleared, cleared, or reconciled. --flag takes a color or
none for unflagged. --memo keeps memos containing the text, ignoring
case, on the transaction or any line. --min and --max bound the signed
amount: outflows are negative, so --max -100 keeps outflows of 100 or
more and --min 0 keeps inflows.

The API serves the request from the most specific of the account, month,
and plan listings, with --since, --until, and one of --unapproved or
--uncategorized applied by the API and the rest applied here. Three
requests: the plans endpoint, the accounts endpoint (which names transfer
targets and resolves --account), then the listing. --category and --payee
each add one request to resolve the name. --csv prints the parent row
followed by one row per split line with the parent's id in parent_id.
To total whole transactions, include only rows with an empty parent_id.

` + planHelp + "\n\n" + dateHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions list --since 2026-08-01 --until 2026-08-31 --csv
  ynab transactions list --account Visa --unapproved
  ynab transactions list --category Groceries --month 2026-08
  ynab transactions list --payee Costco --min -200 --max -50
  ynab transactions list --memo refund --jsonl | jq .amount
  ynab transactions list --jsonl >> history.jsonl`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			filters := transactionFilters{cleared: cleared, flag: flag, memo: memo, Query: register.Query{Unapproved: unapproved, Uncategorized: uncategorized}}
			var err error
			if filters.Since, err = dates.OptionalDate(since, a.deps.now()); err != nil {
				return err
			}
			if filters.Until, err = dates.OptionalDate(until, a.deps.now()); err != nil {
				return err
			}
			if filters.Since != "" && filters.Until != "" && filters.Since > filters.Until {
				return status.Usage(fmt.Sprintf("--since %s is after --until %s", filters.Since, filters.Until))
			}
			if cmd.Flags().Changed("month") {
				if filters.Month, err = dates.MonthInput(month, a.deps.now()); err != nil {
					return err
				}
			}
			if filters.min, err = optionalAmountFlag(minText); err != nil {
				return err
			}
			if filters.max, err = optionalAmountFlag(maxText); err != nil {
				return err
			}

			loaded, client, err := a.connect(cmd)
			if err != nil {
				return err
			}
			plan, err := selection.Plan(cmd.Context(), client, loaded.Config)
			if err != nil {
				return err
			}
			accounts, err := client.Accounts(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}
			if account != "" {
				found, err := selection.Account(account, accounts)
				if err != nil {
					return err
				}
				filters.AccountID = found.ID
			}
			if category != "" {
				groups, err := client.Categories(cmd.Context(), plan.ID)
				if err != nil {
					return err
				}
				found, _, err := selection.Category(category, groups)
				if err != nil {
					return err
				}
				filters.categoryID = found.ID
			}
			if payee != "" {
				payees, err := client.Payees(cmd.Context(), plan.ID)
				if err != nil {
					return err
				}
				found, err := selection.Payee(payee, payees)
				if err != nil {
					return err
				}
				filters.payeeID = found.ID
			}
			transactions, err := filters.Listing().Fetch(cmd.Context(), client, plan.ID, filters.Query)
			if err != nil {
				return err
			}

			if filters.Uncategorized {
				transactions = ynab.TransactionsNeedingCategory(transactions, accounts)
			}
			names := selection.NewAccountNames(accounts)
			var records []transactionRecord
			var total ynab.Amount
			for _, tx := range transactions {
				amount, ok := filters.matching(tx)
				if !ok {
					continue
				}
				total += amount
				records = append(records, newTransactionRecord(tx, names))
			}

			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return render.JSONL(out, records)
			case output.csv:
				return render.CSV(out, csvLines(records))
			}
			if err := render.Table(out, transactionColumns, transactionRows(records, plan.CurrencyFormat)); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, transactionSummary(len(records), total, filters.byName(), plan.CurrencyFormat))
			return err
		}),
	}
	output.bind(command, true)
	command.Flags().StringVar(&account, "account", "", "Account ID or exact name")
	command.Flags().StringVar(&category, "category", "", "Category ID, exact name, or \"Group: Name\"")
	command.Flags().StringVar(&payee, "payee", "", "Payee ID or exact name")
	command.Flags().StringVar(&since, "since", "", "Earliest date: YYYY-MM-DD, today, yesterday")
	command.Flags().StringVar(&until, "until", "", "Latest date: YYYY-MM-DD, today, yesterday")
	command.Flags().StringVar(&month, "month", "", "Month: current, YYYY-MM, YYYY-MM-01")
	command.Flags().BoolVar(&unapproved, "unapproved", false, "Only transactions awaiting approval")
	command.Flags().BoolVar(&uncategorized, "uncategorized", false, "Only transactions without a category")
	bindEnumFlag(command, &cleared, "cleared", "state", "Cleared state: uncleared, cleared, reconciled", clearedStates...)
	bindEnumFlag(command, &flag, "flag", "color", "Flag color, or none for unflagged", flagColors...)
	command.Flags().StringVar(&memo, "memo", "", "Memo contains this text, ignoring case")
	command.Flags().StringVar(&minText, "min", "", "Smallest signed amount, such as -100 or 0")
	command.Flags().StringVar(&maxText, "max", "", "Largest signed amount, such as -50")
	completeFlags(command, a.completeAccounts(), "account")
	completeFlags(command, a.completeCategories(), "category")
	completeFlags(command, a.completePayees(), "payee")
	return command
}

// optionalAmountFlag parses an amount flag at milliunit precision, nil
// when the flag is unset. A bad value is a usage error.
func optionalAmountFlag(text string) (*ynab.Amount, error) {
	if text == "" {
		return nil, nil
	}
	amount, err := ynab.ParseAmount(text, 3)
	if err != nil {
		return nil, status.Usage(err.Error())
	}
	return &amount, nil
}

// containsFold reports whether text contains substring ignoring case.
func containsFold(text, substring string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substring))
}

// transactionSummary is the count line under a transaction table: the
// count and the net total, or the total of the matching lines when a
// category or payee filter selected them.
func transactionSummary(count int, total ynab.Amount, matchingLines bool, currency *ynab.CurrencyFormat) string {
	if matchingLines {
		return fmt.Sprintf("%s, matching lines total %s", render.TransactionCount(count), total.Format(currency))
	}
	return fmt.Sprintf("%s, total %s", render.TransactionCount(count), total.Format(currency))
}
