package cmd

import (
	"fmt"
	"sort"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// reviewRecord is a transaction record plus what it needs: approve,
// categorize, or both.
type reviewRecord struct {
	transactionRecord
	Needs string `json:"needs"`
}

// Values of a review record's needs field.
const (
	needsApprove    = "approve"
	needsBoth       = "both"
	needsCategorize = "categorize"
)

// reviewList merges the unapproved and uncategorized listings, oldest
// first, marking a transaction in both as needing both.
func reviewList(unapproved, uncategorized []ynab.Transaction, accounts accountNames) []reviewRecord {
	var records []reviewRecord
	index := map[string]int{}
	for _, tx := range unapproved {
		index[tx.ID] = len(records)
		records = append(records, reviewRecord{transactionRecord: newTransactionRecord(tx, accounts), Needs: needsApprove})
	}
	for _, tx := range uncategorized {
		if i, ok := index[tx.ID]; ok {
			records[i].Needs = needsBoth
			continue
		}
		records = append(records, reviewRecord{transactionRecord: newTransactionRecord(tx, accounts), Needs: needsCategorize})
	}
	sort.SliceStable(records, func(i, j int) bool { return records[i].Date < records[j].Date })
	return records
}

// reviewSummary counts what needs attention.
func reviewSummary(records []reviewRecord) string {
	if len(records) == 0 {
		return "nothing needs attention"
	}
	approve, categorize := 0, 0
	for _, record := range records {
		if record.Needs != needsCategorize {
			approve++
		}
		if record.Needs != needsApprove {
			categorize++
		}
	}
	verb := "need"
	if len(records) == 1 {
		verb = "needs"
	}
	return fmt.Sprintf("%s %s attention: %d to approve, %d to categorize", transactionCount(len(records)), verb, approve, categorize)
}

var reviewColumns = []column{
	{name: "DATE"}, {name: "ACCOUNT"}, {name: "PAYEE"}, {name: "CATEGORY"}, {name: "MEMO"},
	{name: "AMOUNT", right: true}, {name: "NEEDS"},
}

// reviewRows renders each record as transactionRows does, without the
// cleared, approved, and flag columns and with needs on the parent row.
func reviewRows(records []reviewRecord, currency *ynab.CurrencyFormat) [][]cell {
	var rows [][]cell
	for _, record := range records {
		rows = append(rows, []cell{
			plain(record.Date), plain(record.Account), plain(record.Payee), plain(record.Category), plain(record.Memo),
			plainAmount(record.Amount, currency), plain(record.Needs),
		})
		for _, line := range record.Subtransactions {
			rows = append(rows, []cell{
				plain(""), plain(""), plain(line.Payee), plain("  " + line.Category), plain(line.Memo),
				plainAmount(line.Amount, currency), plain(""),
			})
		}
	}
	return rows
}

// reviewCSVLines flattens splits as csvLines does, repeating needs on
// each line.
func reviewCSVLines(records []reviewRecord) []reviewRecord {
	var rows []reviewRecord
	for _, record := range records {
		for _, line := range csvLines([]transactionRecord{record.transactionRecord}) {
			rows = append(rows, reviewRecord{transactionRecord: line, Needs: record.Needs})
		}
	}
	return rows
}

func newTransactionsReview(a *app) *cobra.Command {
	var output outputFlags
	command := &cobra.Command{
		Use: "review", Short: "List the transactions awaiting approval or a category",
		Long: `List the configured plan's unapproved and uncategorized transactions in
one table, oldest first, with a needs column saying approve, categorize,
or both, and a summary line with the counts. The API's one-year default
window applies. Exit 0 whether or not anything needs attention; the
counts are the payload. Tracking-account transactions and transfers
between plan accounts do not need categories; unapproved ones still
appear for approval. Four requests: the plans endpoint, the accounts
endpoint, then the unapproved and uncategorized listings. --jsonl and
--csv carry the transaction record plus needs; --csv flattens splits as
'transactions list --csv' does.

` + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions review
  ynab transactions review --jsonl | jq -r 'select(.needs == "both") | .id'`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
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
			unapproved, err := client.Transactions(cmd.Context(), plan.ID, ynab.TransactionFilter{Type: ynab.TypeUnapproved})
			if err != nil {
				return err
			}
			uncategorized, err := client.Transactions(cmd.Context(), plan.ID, ynab.TransactionFilter{Type: ynab.TypeUncategorized})
			if err != nil {
				return err
			}

			records := reviewList(unapproved, transactionsNeedingCategory(uncategorized, accounts), newAccountNames(accounts))
			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return writeJSONL(out, records)
			case output.csv:
				return writeCSV(out, reviewCSVLines(records))
			}
			if err := writeTable(out, reviewColumns, reviewRows(records, plan.CurrencyFormat)); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, reviewSummary(records))
			return err
		}),
	}
	output.bind(command, true)
	return command
}
