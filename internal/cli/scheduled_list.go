package cli

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/selection"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

var scheduledColumns = []render.Column{
	{Name: "NEXT DATE"}, {Name: "FREQUENCY"}, {Name: "ACCOUNT"}, {Name: "PAYEE"}, {Name: "CATEGORY"}, {Name: "MEMO"},
	{Name: "AMOUNT", Right: true},
}

// scheduledRows renders each scheduled transaction as a row with a
// split's lines indented beneath it.
func scheduledRows(records []scheduledRecord, currency *ynab.CurrencyFormat) [][]render.Cell {
	var rows [][]render.Cell
	for _, record := range records {
		rows = append(rows, []render.Cell{
			render.Plain(record.NextDate), render.Plain(record.Frequency), render.Plain(record.Account), render.Plain(record.Payee),
			render.Plain(record.Category), render.Plain(record.Memo), render.Amount(record.Amount, currency),
		})
		for _, line := range record.Subtransactions {
			rows = append(rows, []render.Cell{
				render.Plain(""), render.Plain(""), render.Plain(""), render.Plain(line.Payee), render.Plain("  " + line.Category), render.Plain(line.Memo),
				render.Amount(line.Amount, currency),
			})
		}
	}
	return rows
}

func newScheduledList(a *app) *cobra.Command {
	var output outputFlags
	command := &cobra.Command{
		Use: "list", Short: "List the plan's scheduled transactions",
		Long: `List the configured plan's scheduled transactions in the API's order.
Columns: next date, frequency in the API's spelling (such as monthly or
everyOtherWeek), account, payee, category, memo, and amount. A split
shows its lines indented beneath it under the category "Split". Three
requests: the plans endpoint, the accounts endpoint (which names transfer
targets), then the scheduled transactions endpoint. --csv flattens a
split to one row per line with the parent's id in parent_id.

` + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab scheduled list
  ynab scheduled list --jsonl | jq 'select(.frequency == "monthly")'`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
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
			scheduled, err := client.ScheduledTransactions(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}

			names := selection.NewAccountNames(accounts)
			records := make([]scheduledRecord, len(scheduled))
			for i, entry := range scheduled {
				records[i] = newScheduledRecord(entry, names)
			}

			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return render.JSONL(out, records)
			case output.csv:
				return render.CSV(out, scheduledCSVLines(records))
			}
			if err := render.Table(out, scheduledColumns, scheduledRows(records, plan.CurrencyFormat)); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, render.CountNoun(len(records), "scheduled transaction", "scheduled transactions"))
			return err
		}),
	}
	output.bind(command, true)
	return command
}

// scheduledCSVLines flattens splits for CSV as csvLines does for
// transactions.
func scheduledCSVLines(records []scheduledRecord) []scheduledRecord {
	var rows []scheduledRecord
	for _, record := range records {
		parent := record
		parent.Subtransactions = nil
		rows = append(rows, parent)
		for _, line := range record.Subtransactions {
			rows = append(rows, scheduledRecord{
				Account: parent.Account, AccountID: parent.AccountID, FirstDate: parent.FirstDate,
				FlagColor: parent.FlagColor, FlagName: parent.FlagName, Frequency: parent.Frequency, NextDate: parent.NextDate,
				Amount: line.Amount, Category: line.Category, CategoryID: line.CategoryID, ID: line.ID,
				Memo: line.Memo, ParentID: parent.ID, Payee: line.Payee, PayeeID: line.PayeeID,
				TransferAccount: line.TransferAccount, TransferAccountID: line.TransferAccountID,
			})
		}
	}
	return rows
}
