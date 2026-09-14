package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMonthsList(a *app) *cobra.Command {
	var output outputFlags
	command := &cobra.Command{
		Use: "list", Short: "List every plan month with its totals",
		Long: `List every month of the configured plan in the API's order, oldest first,
including months the plan has not reached yet. Columns: income, assigned,
activity, ready to assign, and age of money in days (blank until the plan
has one). Negative ready to assign is red when color is on. Two requests:
the plans endpoint, then the months endpoint.

` + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab months list
  ynab months list --jsonl | jq 'select(.ready_to_assign < 0)'
  ynab months list --csv > months.csv`,
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
			months, err := client.Months(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}

			records := make([]monthRecord, len(months))
			for i, month := range months {
				records[i] = newMonthRecord(month)
			}

			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return writeJSONL(out, records)
			case output.csv:
				return writeCSV(out, records)
			}
			columns := []column{
				{name: "MONTH"},
				{name: "INCOME", right: true}, {name: "ASSIGNED", right: true}, {name: "ACTIVITY", right: true},
				{name: "READY TO ASSIGN", right: true}, {name: "AGE OF MONEY", right: true},
			}
			rows := make([][]cell, len(records))
			colors := a.palette(cmd)
			for i, record := range records {
				rows[i] = []cell{
					plain(record.Month),
					plainAmount(record.Income, plan.CurrencyFormat), plainAmount(record.Assigned, plan.CurrencyFormat), plainAmount(record.Activity, plan.CurrencyFormat),
					availableCell(record.ReadyToAssign, plan.CurrencyFormat, colors), plain(optionalInt(record.AgeOfMoney)),
				}
			}
			if err := writeTable(out, columns, rows); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, countNoun(len(records), "month", "months"))
			return err
		}),
	}
	output.bind(command, true)
	return command
}
