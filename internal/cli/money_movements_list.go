package cli

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/input"
	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/selection"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newMoneyMovementsList(a *app) *cobra.Command {
	var output outputFlags
	var month string
	command := &cobra.Command{
		Use: "list", Short: "List the plan's money movements, all or for one month",
		Long: `List the configured plan's money movements in the API's order. Columns:
month, when the move was made, the source and target categories ("Ready
to Assign" when the API records none), the amount, and the note. Without
--month every movement is listed; --month lists one month's. Three
requests: the plans endpoint, the categories endpoint (which names the
categories), then the money movements endpoint.

` + planHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab money-movements list
  ynab money-movements list --month current
  ynab money-movements list --month 2026-08 --csv > moves.csv`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			apiMonth := ""
			if cmd.Flags().Changed("month") {
				var err error
				if apiMonth, err = input.Month(month, a.deps.now()); err != nil {
					return err
				}
			}
			loaded, client, err := a.connect(cmd)
			if err != nil {
				return err
			}
			plan, err := selection.Plan(cmd.Context(), client, loaded.Config)
			if err != nil {
				return err
			}
			groups, err := client.Categories(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}
			var movements []ynab.MoneyMovement
			if apiMonth != "" {
				movements, err = client.MonthMoneyMovements(cmd.Context(), plan.ID, apiMonth)
			} else {
				movements, err = client.MoneyMovements(cmd.Context(), plan.ID)
			}
			if err != nil {
				return err
			}

			categories := newCategoryNames(groups)
			records := make([]moneyMovementRecord, len(movements))
			for i, movement := range movements {
				records[i] = newMoneyMovementRecord(movement, categories)
			}

			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return render.JSONL(out, records)
			case output.csv:
				return render.CSV(out, records)
			}
			columns := []render.Column{{Name: "MONTH"}, {Name: "MOVED AT"}, {Name: "FROM"}, {Name: "TO"}, {Name: "AMOUNT", Right: true}, {Name: "NOTE"}}
			rows := make([][]render.Cell, len(records))
			for i, record := range records {
				rows[i] = []render.Cell{
					render.Plain(record.Month), render.Plain(render.OptionalTime(record.MovedAt)), render.Plain(record.From), render.Plain(record.To),
					render.Amount(record.Amount, plan.CurrencyFormat), render.Plain(record.Note),
				}
			}
			if err := render.Table(out, columns, rows); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, render.CountNoun(len(records), "money movement", "money movements"))
			return err
		}),
	}
	output.bind(command, true)
	command.Flags().StringVar(&month, "month", "", "Month: current, YYYY-MM, YYYY-MM-01")
	return command
}
