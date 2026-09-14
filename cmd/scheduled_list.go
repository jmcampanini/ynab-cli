package cmd

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

var scheduledColumns = []column{
	{name: "NEXT DATE"}, {name: "FREQUENCY"}, {name: "ACCOUNT"}, {name: "PAYEE"}, {name: "CATEGORY"}, {name: "MEMO"},
	{name: "AMOUNT", right: true},
}

// scheduledRows renders each scheduled transaction as a row with a
// split's lines indented beneath it.
func scheduledRows(records []scheduledRecord, currency *ynab.CurrencyFormat) [][]cell {
	var rows [][]cell
	for _, record := range records {
		rows = append(rows, []cell{
			plain(record.NextDate), plain(record.Frequency), plain(record.Account), plain(record.Payee),
			plain(record.Category), plain(record.Memo), plainAmount(record.Amount, currency),
		})
		for _, line := range record.Subtransactions {
			rows = append(rows, []cell{
				plain(""), plain(""), plain(""), plain(line.Payee), plain("  " + line.Category), plain(line.Memo),
				plainAmount(line.Amount, currency),
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
			plan, err := selectedPlan(cmd.Context(), client, loaded.Config)
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

			names := newAccountNames(accounts)
			records := make([]scheduledRecord, len(scheduled))
			for i, entry := range scheduled {
				records[i] = newScheduledRecord(entry, names)
			}

			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return writeJSONL(out, records)
			case output.csv:
				return writeCSV(out, scheduledCSVLines(records))
			}
			if err := writeTable(out, scheduledColumns, scheduledRows(records, plan.CurrencyFormat)); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, countNoun(len(records), "scheduled transaction", "scheduled transactions"))
			return err
		}),
	}
	output.bind(command, true)
	return command
}
