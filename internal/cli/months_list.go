package cli

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/selection"
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
			plan, err := selection.Plan(cmd.Context(), client, loaded.Config)
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
				return render.JSONL(out, records)
			case output.csv:
				return render.CSV(out, records)
			}
			columns := []render.Column{
				{Name: "MONTH"},
				{Name: "INCOME", Right: true}, {Name: "ASSIGNED", Right: true}, {Name: "ACTIVITY", Right: true},
				{Name: "READY TO ASSIGN", Right: true}, {Name: "AGE OF MONEY", Right: true},
			}
			rows := make([][]render.Cell, len(records))
			colors := a.palette(cmd)
			for i, record := range records {
				rows[i] = []render.Cell{
					render.Plain(record.Month),
					render.Amount(record.Income, plan.CurrencyFormat), render.Amount(record.Assigned, plan.CurrencyFormat), render.Amount(record.Activity, plan.CurrencyFormat),
					render.Available(record.ReadyToAssign, plan.CurrencyFormat, colors), render.Plain(render.OptionalInt(record.AgeOfMoney)),
				}
			}
			if err := render.Table(out, columns, rows); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, render.CountNoun(len(records), "month", "months"))
			return err
		}),
	}
	output.bind(command, true)
	return command
}
