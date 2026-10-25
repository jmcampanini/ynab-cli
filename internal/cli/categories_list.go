package cli

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/categoryview"
	"github.com/jmcampanini/ynab-cli/internal/dates"
	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/selection"
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
amount still needed this month to stay on track. For a "plan your
spending" target in a future month, the API's underfunded amount counts
funding from earlier periods, which the YNAB app ignores, so the two can
differ. Hidden categories and the categories of hidden groups are left out
unless --hidden is given, which shows them faint with a "(hidden)" suffix.
Internal groups, such as Credit Card Payments, are shown. Negative
available is red when color is on. Without --month, two requests: the
plans endpoint, then the categories endpoint, which carries the current
month. With --month, a third request fetches that month.

` + planHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab categories list
  ynab categories list --month 2026-08
  ynab categories list --hidden
  ynab categories list --jsonl | jq 'select(.available < 0)'
  ynab categories list --csv > categories.csv`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			apiMonth, err := dates.MonthInput(monthText, a.deps.now())
			if err != nil {
				return err
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
			var monthValues map[string]ynab.Category
			if cmd.Flags().Changed("month") {
				month, err := client.Month(cmd.Context(), plan.ID, apiMonth)
				if err != nil {
					return err
				}
				monthValues = ynab.CategoriesByID(month.Categories)
			}

			listing, err := categoryview.List(groups, monthValues, apiMonth, includeHidden)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return render.JSONL(out, listing.Records())
			case output.csv:
				return render.CSV(out, listing.Records())
			}
			if err := render.Table(out, categoryview.Columns, listing.Rows(plan.CurrencyFormat, a.palette(cmd))); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, listing.Summary())
			return err
		}),
	}
	output.bind(command, true)
	bindMonthFlag(command, &monthText)
	command.Flags().BoolVar(&includeHidden, "hidden", false, "Include hidden categories and groups")
	return command
}
