package cmd

import (
	"strconv"
	"time"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
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
month. needs_whole_amount, on NEED targets, is true to set aside the full
amount each period and false to refill up to it.

` + planHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab categories get Internet
  ynab categories get "Bills: Internet" --month 2026-08
  ynab categories get 7a3e... --jsonl`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
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
			category, group, err := findCategory(args[0], groups)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("month") {
				category, err = client.MonthCategory(cmd.Context(), plan.ID, apiMonth, category.ID)
				if err != nil {
					return err
				}
			}

			record := newCategoryRecord(category, group, apiMonth)
			out := cmd.OutOrStdout()
			if output.jsonl {
				return writeJSONL(out, []categoryRecord{record})
			}
			return writeFields(out, categoryFields(record, plan.CurrencyFormat))
		}),
	}
	output.bind(command, false)
	bindMonthFlag(command, &monthText)
	return command
}

// categoryFields lists the record for human output, decoding the target
// to words and showing the target rows only when there is one.
func categoryFields(record categoryRecord, currency *ynab.CurrencyFormat) [][2]string {
	fields := [][2]string{
		{"id", record.ID},
		{"name", record.Name},
		{"group", record.Group},
		{"month", record.Month},
		{"hidden", strconv.FormatBool(record.Hidden)},
		{"internal", strconv.FormatBool(record.Internal)},
		{"note", record.Note},
		{"assigned", record.Assigned.Format(currency)},
		{"activity", record.Activity.Format(currency)},
		{"available", record.Available.Format(currency)},
	}
	target := record.Target
	if target == nil {
		return append(fields, [2]string{"target", "none"})
	}
	return append(fields,
		[2]string{"target", target.typeWords()},
		[2]string{"target amount", target.Amount.Format(currency)},
		[2]string{"target date", target.Date},
		[2]string{"target cadence", target.cadenceWords()},
		[2]string{"target day", target.dayWords()},
		[2]string{"target rollover", target.rolloverWords()},
		[2]string{"target created", target.CreatedMonth},
		[2]string{"target progress", optionalPercent(target.PercentComplete)},
		[2]string{"target months left", optionalInt(target.MonthsToAssign)},
		[2]string{"target underfunded", optionalAmount(target.Underfunded, currency)},
		[2]string{"target funded", optionalAmount(target.OverallFunded, currency)},
		[2]string{"target left", optionalAmount(target.OverallLeft, currency)},
		[2]string{"target snoozed", optionalTime(target.SnoozedAt)},
	)
}

func optionalAmount(amount *ynab.Amount, currency *ynab.CurrencyFormat) string {
	if amount == nil {
		return ""
	}
	return amount.Format(currency)
}

func optionalPercent(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value) + "%"
}

func optionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.DateTime)
}
