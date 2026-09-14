package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMonthsGet(a *app) *cobra.Command {
	var output outputFlags
	command := &cobra.Command{
		Use: "get [MONTH]", Short: "Show one month's totals and its category rows",
		Long: `Show one month of the configured plan: its income, assigned, activity,
ready to assign, and age of money, then its category rows grouped and
subtotaled as 'ynab categories list --month MONTH' shows them, without
hidden categories. MONTH defaults to current. Three requests: the plans
endpoint, the categories endpoint for the group order, then the month.
--jsonl prints one object with the month's totals; the category rows come
from 'ynab categories list --month MONTH --jsonl'.

` + planHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab months get
  ynab months get 2026-08
  ynab months get --jsonl | jq .ready_to_assign`,
		Args: cobra.MaximumNArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			monthText := "current"
			if len(args) == 1 {
				monthText = args[0]
			}
			apiMonth, err := a.month(monthText)
			if err != nil {
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
			groups, err := client.Categories(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}
			month, err := client.Month(cmd.Context(), plan.ID, apiMonth)
			if err != nil {
				return err
			}

			record := newMonthRecord(month)
			listing, err := listCategories(groups, categoriesByID(month.Categories), month.Month, false)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if output.jsonl {
				return writeJSONL(out, []monthRecord{record})
			}
			currency := plan.CurrencyFormat
			err = writeFields(out, [][2]string{
				{"month", record.Month},
				{"note", record.Note},
				{"income", record.Income.Format(currency)},
				{"assigned", record.Assigned.Format(currency)},
				{"activity", record.Activity.Format(currency)},
				{"ready to assign", record.ReadyToAssign.Format(currency)},
				{"age of money", optionalInt(record.AgeOfMoney)},
			})
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			if err := writeTable(out, categoryColumns, listing.rows(currency, a.palette(cmd))); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, listing.summary())
			return err
		}),
	}
	output.bind(command, false)
	return command
}
