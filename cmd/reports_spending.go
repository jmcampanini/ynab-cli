package cmd

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// Values of --by.
const (
	byCategory = "category"
	byPayee    = "payee"
)

// Row names for amounts without a category or a payee.
const (
	uncategorizedRow = "Uncategorized"
	noPayeeRow       = "No payee"
)

// spendingRecord is the --jsonl and --csv shape of one row of the
// spending report: one category or one payee over the range. ID is empty
// and Group absent on the row for uncategorized amounts; Group is absent
// under --by payee.
type spendingRecord struct {
	ID           string      `json:"id,omitempty"`
	Name         string      `json:"name"`
	Group        string      `json:"group,omitempty"`
	Outflows     ynab.Amount `json:"outflows"`
	Inflows      ynab.Amount `json:"inflows"`
	Net          ynab.Amount `json:"net"`
	Transactions int         `json:"transactions"`
}

// spendingLine is one categorized amount of a transaction: the
// transaction itself, or one line of a split.
type spendingLine struct {
	amount     ynab.Amount
	categoryID string
	category   string
	payeeID    string
	payee      string
}

// spendingLines breaks a transaction into the amounts the report totals,
// leaving out transfers between two on-plan accounts, whole or as a
// split line, since those move money without spending it. A split's
// lines carry their own category and payee, inheriting the parent's
// payee when they name none.
func spendingLines(tx ynab.Transaction, onPlan map[string]bool) []spendingLine {
	transfer := func(target *string) bool {
		return target != nil && onPlan[tx.AccountID] && onPlan[*target]
	}
	if len(tx.Subtransactions) == 0 {
		if transfer(tx.TransferAccountID) {
			return nil
		}
		return []spendingLine{{amount: tx.Amount, categoryID: text(tx.CategoryID), category: text(tx.CategoryName), payeeID: text(tx.PayeeID), payee: text(tx.PayeeName)}}
	}

	var lines []spendingLine
	for _, line := range tx.Subtransactions {
		if transfer(line.TransferAccountID) {
			continue
		}
		payeeID, payee := line.PayeeID, line.PayeeName
		if payeeID == nil {
			payeeID, payee = tx.PayeeID, tx.PayeeName
		}
		lines = append(lines, spendingLine{amount: line.Amount, categoryID: text(line.CategoryID), category: text(line.CategoryName), payeeID: text(payeeID), payee: text(payee)})
	}
	return lines
}

// spendingReport is the rows of the spending report with its totals.
type spendingReport struct {
	by           string
	inflows      ynab.Amount
	outflows     ynab.Amount
	records      []spendingRecord
	transactions int
}

// newSpendingReport totals the transactions by category or payee, sorted
// by net with the largest outflow first, ties by name then ID. A transaction
// counts once on each row it touches and once in the total. groupNames
// maps category IDs to group names for the category view.
func newSpendingReport(by string, transactions []ynab.Transaction, onPlan map[string]bool, groupNames map[string]string) spendingReport {
	report := spendingReport{by: by}
	rows := map[string]*spendingRecord{}
	counted := map[string]map[string]bool{}
	for _, tx := range transactions {
		lines := spendingLines(tx, onPlan)
		if len(lines) == 0 {
			continue
		}
		report.transactions++
		for _, line := range lines {
			id, name := line.categoryID, line.category
			if by == byPayee {
				id, name = line.payeeID, line.payee
			}
			row, ok := rows[id]
			if !ok {
				row = &spendingRecord{ID: id, Name: name}
				switch {
				case by == byPayee && id == "":
					row.Name = noPayeeRow
				case by == byCategory && id == "":
					row.Name = uncategorizedRow
				case by == byCategory:
					row.Group = groupNames[id]
				}
				rows[id] = row
				counted[id] = map[string]bool{}
			}
			if line.amount < 0 {
				row.Outflows += line.amount
				report.outflows += line.amount
			} else {
				row.Inflows += line.amount
				report.inflows += line.amount
			}
			row.Net += line.amount
			if !counted[id][tx.ID] {
				counted[id][tx.ID] = true
				row.Transactions++
			}
		}
	}

	for _, row := range rows {
		report.records = append(report.records, *row)
	}
	slices.SortFunc(report.records, func(a, b spendingRecord) int {
		return cmp.Or(cmp.Compare(a.Net, b.Net), cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID))
	})
	return report
}

// columns are the table's columns for the chosen grouping.
func (r spendingReport) columns() []column {
	amounts := []column{{name: "OUTFLOWS", right: true}, {name: "INFLOWS", right: true}, {name: "NET", right: true}, {name: "TRANSACTIONS", right: true}}
	if r.by == byPayee {
		return append([]column{{name: "PAYEE"}}, amounts...)
	}
	return append([]column{{name: "GROUP"}, {name: "CATEGORY"}}, amounts...)
}

func (r spendingReport) rows(currency *ynab.CurrencyFormat) [][]cell {
	rows := make([][]cell, len(r.records))
	for i, record := range r.records {
		amounts := []cell{plainAmount(record.Outflows, currency), plainAmount(record.Inflows, currency), plainAmount(record.Net, currency), plain(fmt.Sprint(record.Transactions))}
		if r.by == byPayee {
			rows[i] = append([]cell{plain(record.Name)}, amounts...)
			continue
		}
		rows[i] = append([]cell{plain(record.Group), plain(record.Name)}, amounts...)
	}
	return rows
}

// summary is the total line under the table.
func (r spendingReport) summary(rangeWords string, currency *ynab.CurrencyFormat) string {
	rows := countNoun(len(r.records), "category", "categories")
	if r.by == byPayee {
		rows = countNoun(len(r.records), "payee", "payees")
	}
	return fmt.Sprintf("%s, %s %s: outflows %s, inflows %s, net %s", rows, transactionCount(r.transactions), rangeWords,
		r.outflows.Format(currency), r.inflows.Format(currency), (r.outflows + r.inflows).Format(currency))
}

// rangeWords describes the date range the report covers: the month, or
// the dates, of which at least one is set.
func rangeWords(apiMonth, since, until string) string {
	switch {
	case apiMonth != "":
		return "in " + shortMonth(apiMonth)
	case since != "" && until != "":
		return "from " + since + " to " + until
	case since != "":
		return "since " + since
	}
	return "through " + until
}

func newReportsSpending(a *app) *cobra.Command {
	var output outputFlags
	var by, since, until, month, account string
	command := &cobra.Command{
		Use: "spending [--by category|payee] [--since D] [--until D] [--month M] [--account A]", Short: "Total outflows and inflows by category or payee",
		Long: `Total the outflows and inflows of the configured plan's transactions
over a date range, one row per category (with its group) or, with --by
payee, per payee, sorted by net with the largest outflow first. Each row
carries the count of transactions touching it, and the total line
carries the counts and totals of the whole range. A split's lines count
toward their own categories and payees, a line without a payee taking
the parent's. Transfers between two on-plan accounts are left out, as
whole transactions or as split lines, since they move money without
spending it; transfers to tracking accounts count under their category.
Amounts without a category or payee gather on an "Uncategorized" or "No
payee" row.

Without --since, --until, or --month the range is the current month.
--month is that month's range and cannot combine with --since or
--until. With --since alone the range has no end, and with --until
alone it starts one year back, the API's default window. --account
narrows the range to one account by ID or exact name. Three requests:
the plans endpoint, the accounts endpoint (which marks on-plan accounts
and resolves --account), then the listing, served from the most specific
of the account, month, and plan listings; --by category adds one for the
categories endpoint, which supplies the group names.

` + planHelp + "\n\n" + dateHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab reports spending
  ynab reports spending --by payee --since 2026-01-01 --csv > payees.csv
  ynab reports spending --month 2026-08 --account Visa
  ynab reports spending --jsonl | jq 'select(.net < -100)'`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			filters := transactionFilters{}
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
			if filters.since == "" && filters.until == "" {
				if filters.month, err = a.month(month); err != nil {
					return err
				}
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
			groupNames := map[string]string{}
			if by == byCategory {
				groups, err := client.Categories(cmd.Context(), plan.ID)
				if err != nil {
					return err
				}
				for _, group := range groups {
					for _, category := range group.Categories {
						groupNames[category.ID] = group.Name
					}
				}
			}
			transactions, err := filters.listing().fetch(cmd.Context(), client, plan.ID, filters)
			if err != nil {
				return err
			}

			onPlan := make(map[string]bool, len(accounts))
			for _, account := range accounts {
				onPlan[account.ID] = account.OnPlan
			}
			report := newSpendingReport(by, transactions, onPlan, groupNames)
			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return writeJSONL(out, report.records)
			case output.csv:
				return writeCSV(out, report.records)
			}
			if err := writeTable(out, report.columns(), report.rows(plan.CurrencyFormat)); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, report.summary(rangeWords(filters.month, filters.since, filters.until), plan.CurrencyFormat))
			return err
		}),
	}
	output.bind(command, true)
	by = byCategory
	bindEnumFlag(command, &by, "by", "grouping", "Group rows by category or payee", byCategory, byPayee)
	command.Flags().StringVar(&since, "since", "", "Earliest date: YYYY-MM-DD, today, yesterday")
	command.Flags().StringVar(&until, "until", "", "Latest date: YYYY-MM-DD, today, yesterday")
	bindMonthFlag(command, &month)
	command.Flags().StringVar(&account, "account", "", "Account ID or exact name")
	command.MarkFlagsMutuallyExclusive("month", "since")
	command.MarkFlagsMutuallyExclusive("month", "until")
	completeFlags(command, a.completeAccounts(), "account")
	return command
}
