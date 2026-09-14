package cmd

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newCategoriesList(a *app) *cobra.Command {
	var output outputFlags
	var monthText string
	var includeHidden bool
	command := &cobra.Command{
		Use: "list", Short: "List the plan's categories by group for one month",
		Long: `List the configured plan's categories grouped by category group in the
plan's order. Each group row carries the subtotals of the categories shown
beneath it. Columns: assigned, activity, available, the target as its type
code and amount (see 'ynab categories get --help' for the codes), and the
amount still needed this month to stay on track. Hidden categories and the
categories of hidden groups are left out unless --hidden is given, which
shows them faint with a "(hidden)" suffix. Internal groups, such as Credit
Card Payments, are shown. Negative available is red when color is on.
Without --month, two requests: the plans endpoint, then the categories
endpoint, which carries the current month. With --month, a third request
fetches that month.

` + planHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab categories list
  ynab categories list --month 2026-08
  ynab categories list --hidden
  ynab categories list --jsonl | jq 'select(.available < 0)'
  ynab categories list --csv > categories.csv`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
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
			var monthValues map[string]ynab.Category
			if cmd.Flags().Changed("month") {
				month, err := client.Month(cmd.Context(), plan.ID, apiMonth)
				if err != nil {
					return err
				}
				monthValues = categoriesByID(month.Categories)
			}

			listing, err := listCategories(groups, monthValues, apiMonth, includeHidden)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return writeJSONL(out, listing.records())
			case output.csv:
				return writeCSV(out, listing.records())
			}
			if err := writeTable(out, categoryColumns, listing.rows(plan.CurrencyFormat, a.palette(cmd))); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, listing.summary())
			return err
		}),
	}
	output.bind(command, true)
	bindMonthFlag(command, &monthText)
	command.Flags().BoolVar(&includeHidden, "hidden", false, "Include hidden categories and groups")
	return command
}

func categoriesByID(categories []ynab.Category) map[string]ynab.Category {
	byID := make(map[string]ynab.Category, len(categories))
	for _, category := range categories {
		byID[category.ID] = category
	}
	return byID
}
