package cli

import (
	"github.com/jmcampanini/ynab-cli/internal/categoryview"
	"github.com/jmcampanini/ynab-cli/internal/input"
	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/selection"
	"github.com/spf13/cobra"
)

func newCategoriesGet(a *app) *cobra.Command {
	var output outputFlags
	var monthText string
	command := &cobra.Command{
		Use: "get CATEGORY", Short: "Show every field of one category for one month",
		Long: `Show one category of the configured plan with its amounts for one month
and its target decoded to words. CATEGORY is the category ID, its exact
name, or "Group: Name", matched case-insensitively; hidden categories match
too. No prefix or fuzzy matching: a miss fails and lists the closest names,
and a bare name that exists in more than one group fails and lists each
"Group: Name" form. Two requests: the plans endpoint, then the categories
endpoint; --month adds a third for that month's amounts. --jsonl prints one
object whose target keeps the API's codes.

Target type codes: TB target balance, TBD target balance by date, MF
monthly funding, NEED plan your spending, DEBT debt payoff. Cadence codes:
0 none, 1 monthly, 2 weekly, 13 yearly, each repeating every
cadence_frequency units; 3 to 12 every 2 to 11 months; 14 every 2 years.
day is a weekday (0 Sunday) for weekly targets, otherwise a day of the
month; null on a repeating non-weekly target means the last day of the
month. needs_whole_amount, on NEED targets, is true to set aside the full
amount each period and false to refill up to it.

` + futureRefillHelp + "\n\n" + planHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab categories get Internet
  ynab categories get "Bills: Internet" --month 2026-08
  ynab categories get 7a3e... --jsonl`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			apiMonth, err := input.Month(monthText, a.deps.now())
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
			category, group, err := selection.Category(args[0], groups)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("month") {
				category, err = client.MonthCategory(cmd.Context(), plan.ID, apiMonth, category.ID)
				if err != nil {
					return err
				}
			}

			record := categoryview.NewRecord(category, group, apiMonth)
			a.warnFutureRefill(cmd, []categoryview.Record{record}, plan.CurrencyFormat)
			out := cmd.OutOrStdout()
			if output.jsonl {
				return render.JSONL(out, []categoryview.Record{record})
			}
			return render.Fields(out, categoryFields(record, plan.CurrencyFormat))
		}),
	}
	output.bind(command, false)
	bindMonthFlag(command, &monthText)
	command.ValidArgsFunction = a.completeCategories()
	return command
}
