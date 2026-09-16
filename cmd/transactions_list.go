package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/jmcampanini/ynab-cli/internal/dates"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// Flag values for --cleared and --flag.
var (
	clearedStates = []string{"uncleared", "cleared", "reconciled"}
	flagColors    = []string{"red", "orange", "yellow", "green", "blue", "purple", "none"}
)

// transactionFilters is every list filter after name resolution and date
// parsing. Empty strings and nil amounts mean the filter is off.
type transactionFilters struct {
	accountID     string
	categoryID    string
	cleared       string
	flag          string
	max           *ynab.Amount
	memo          string
	min           *ynab.Amount
	month         string
	payeeID       string
	since         string
	unapproved    bool
	uncategorized bool
	until         string
}

// listingKind names the endpoint a filter set is served from.
type listingKind string

const (
	listAccount listingKind = "account"
	listMonth   listingKind = "month"
	listPlan    listingKind = "plan"
)

// listing is the endpoint choice: the most specific of the account, month,
// and plan listings, plus the query the API applies. When the account
// endpoint wins, --month narrows the dates instead.
type listing struct {
	kind  listingKind
	query ynab.TransactionFilter
}

// listing picks the endpoint and the API-side query. The API accepts one
// type; uncategorized goes to the API because its rule about transfers is
// the server's, and unapproved stays client-side when both are set.
func (f transactionFilters) listing() listing {
	query := ynab.TransactionFilter{SinceDate: f.since, UntilDate: f.until}
	switch {
	case f.uncategorized:
		query.Type = ynab.TypeUncategorized
	case f.unapproved:
		query.Type = ynab.TypeUnapproved
	}
	switch {
	case f.accountID != "" && f.month != "":
		start, end := monthBounds(f.month)
		if query.SinceDate == "" || query.SinceDate < start {
			query.SinceDate = start
		}
		if query.UntilDate == "" || query.UntilDate > end {
			query.UntilDate = end
		}
		return listing{kind: listAccount, query: query}
	case f.accountID != "":
		return listing{kind: listAccount, query: query}
	case f.month != "":
		return listing{kind: listMonth, query: query}
	}
	return listing{kind: listPlan, query: query}
}

// monthBounds returns the first and last dates of an API month.
func monthBounds(apiMonth string) (string, string) {
	start, err := time.Parse(dates.APIMonth, apiMonth)
	if err != nil {
		return apiMonth, apiMonth
	}
	return start.Format(time.DateOnly), start.AddDate(0, 1, -1).Format(time.DateOnly)
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
	if f.unapproved && tx.Approved ||
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
	color := text(tx.FlagColor)
	if f.flag == "none" {
		return color == ""
	}
	return color == f.flag
}

func memoMatches(tx ynab.Transaction, memo string) bool {
	if containsFold(text(tx.Memo), memo) {
		return true
	}
	for _, line := range tx.Subtransactions {
		if containsFold(text(line.Memo), memo) {
			return true
		}
	}
	return false
}

// fetch performs the chosen listing.
func (l listing) fetch(ctx context.Context, client *ynab.Client, planID string, f transactionFilters) ([]ynab.Transaction, error) {
	switch l.kind {
	case listAccount:
		return client.AccountTransactions(ctx, planID, f.accountID, l.query)
	case listMonth:
		return client.MonthTransactions(ctx, planID, f.month, l.query)
	}
	return client.Transactions(ctx, planID, l.query)
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
each add one request to resolve the name. --csv flattens a split to one
row per line with the parent's id in parent_id.

` + planHelp + "\n\n" + dateHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions list --since 2026-08-01 --until 2026-08-31 --csv
  ynab transactions list --account Visa --unapproved
  ynab transactions list --category Groceries --month 2026-08
  ynab transactions list --payee Costco --min -200 --max -50
  ynab transactions list --memo refund --jsonl | jq .amount
  ynab transactions list --jsonl >> history.jsonl`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			filters := transactionFilters{cleared: cleared, flag: flag, memo: memo, unapproved: unapproved, uncategorized: uncategorized}
			var err error
			if filters.since, err = optionalDate(a, since); err != nil {
				return err
			}
			if filters.until, err = optionalDate(a, until); err != nil {
				return err
			}
			if filters.since != "" && filters.until != "" && filters.since > filters.until {
				return usageError(fmt.Sprintf("--since %s is after --until %s", filters.since, filters.until))
			}
			if cmd.Flags().Changed("month") {
				if filters.month, err = a.month(month); err != nil {
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
			plan, err := selectedPlan(cmd.Context(), client, loaded.Config)
			if err != nil {
				return err
			}
			accounts, err := client.Accounts(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}
			if account != "" {
				found, err := findAccount(account, accounts)
				if err != nil {
					return err
				}
				filters.accountID = found.ID
			}
			if category != "" {
				groups, err := client.Categories(cmd.Context(), plan.ID)
				if err != nil {
					return err
				}
				found, _, err := findCategory(category, groups)
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
				found, err := findPayee(payee, payees)
				if err != nil {
					return err
				}
				filters.payeeID = found.ID
			}
			transactions, err := filters.listing().fetch(cmd.Context(), client, plan.ID, filters)
			if err != nil {
				return err
			}

			names := newAccountNames(accounts)
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
				return writeJSONL(out, records)
			case output.csv:
				return writeCSV(out, csvLines(records))
			}
			if err := writeTable(out, transactionColumns, transactionRows(records, plan.CurrencyFormat)); err != nil {
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

// optionalDate parses a date flag, blank when the flag is unset.
func optionalDate(a *app, text string) (string, error) {
	if text == "" {
		return "", nil
	}
	return a.date(text)
}

// optionalAmountFlag parses an amount flag at milliunit precision, nil
// when the flag is unset. A bad value is a usage error.
func optionalAmountFlag(text string) (*ynab.Amount, error) {
	if text == "" {
		return nil, nil
	}
	amount, err := ynab.ParseAmount(text, 3)
	if err != nil {
		return nil, usageError(err.Error())
	}
	return &amount, nil
}
