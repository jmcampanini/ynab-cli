package cli

import (
	"fmt"
	"strconv"

	"github.com/jmcampanini/ynab-cli/internal/categoryview"
	"github.com/jmcampanini/ynab-cli/internal/dates"
	"github.com/jmcampanini/ynab-cli/internal/input"
	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/selection"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// fundingRecord is the --jsonl and --csv shape of one target's funding in
// one month. TargetType keeps the API's code, as the category record
// does; the table spells it out.
type fundingRecord struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Group           string       `json:"group"`
	Month           string       `json:"month"`
	TargetType      string       `json:"target_type"`
	TargetAmount    ynab.Amount  `json:"target_amount"`
	Assigned        ynab.Amount  `json:"assigned"`
	Activity        ynab.Amount  `json:"activity"`
	Available       ynab.Amount  `json:"available"`
	Underfunded     *ynab.Amount `json:"underfunded,omitempty"`
	PercentComplete *int         `json:"percent_complete,omitempty"`
	MonthsToAssign  *int         `json:"months_to_assign,omitempty"`
}

// fundingReport is the rows of the funding report with the counts the
// summary states: every target, and the underfunded ones with their
// total, whether or not the rows were filtered to them.
type fundingReport struct {
	records          []fundingRecord
	targets          int
	underfundedCount int
	underfundedTotal ynab.Amount
}

// newFundingReport builds one row per category with a target, in plan
// order, over the categories 'categories list' shows: hidden categories
// and the API's internal ones are left out. With underfundedOnly, only
// rows whose target still needs money are kept. Rows whose target
// carries no underfunded amount count as needing nothing.
func newFundingReport(groups []ynab.CategoryGroup, month ynab.Month, underfundedOnly bool) (fundingReport, error) {
	listing, err := categoryview.List(groups, ynab.CategoriesByID(month.Categories), month.Month, false)
	if err != nil {
		return fundingReport{}, err
	}

	var report fundingReport
	for _, category := range listing.Records() {
		if category.Target == nil || category.Internal {
			continue
		}
		report.targets++
		underfunded := category.Target.Underfunded != nil && *category.Target.Underfunded > 0
		if underfunded {
			report.underfundedCount++
			report.underfundedTotal += *category.Target.Underfunded
		}
		if underfundedOnly && !underfunded {
			continue
		}
		report.records = append(report.records, fundingRecord{
			Activity:        category.Activity,
			Assigned:        category.Assigned,
			Available:       category.Available,
			Group:           category.Group,
			ID:              category.ID,
			Month:           category.Month,
			MonthsToAssign:  category.Target.MonthsToAssign,
			Name:            category.Name,
			PercentComplete: category.Target.PercentComplete,
			TargetAmount:    category.Target.Amount,
			TargetType:      category.Target.Type,
			Underfunded:     category.Target.Underfunded,
		})
	}
	return report, nil
}

var fundingColumns = []render.Column{
	{Name: "GROUP"}, {Name: "NAME"}, {Name: "TARGET TYPE"}, {Name: "TARGET", Right: true},
	{Name: "ASSIGNED", Right: true}, {Name: "ACTIVITY", Right: true}, {Name: "AVAILABLE", Right: true},
	{Name: "UNDERFUNDED", Right: true}, {Name: "COMPLETE", Right: true}, {Name: "MONTHS LEFT", Right: true},
}

// rows renders the report with negative available and positive
// underfunded amounts in red.
func (r fundingReport) rows(currency *ynab.CurrencyFormat, colors render.Palette) [][]render.Cell {
	rows := make([][]render.Cell, len(r.records))
	for i, record := range r.records {
		words, ok := categoryview.TargetTypeWords[record.TargetType]
		if !ok {
			words = record.TargetType
		}
		underfunded := render.Plain("")
		if record.Underfunded != nil {
			underfunded = render.Amount(*record.Underfunded, currency)
			if *record.Underfunded > 0 {
				underfunded.Paint = colors.Red
			}
		}
		complete := ""
		if record.PercentComplete != nil {
			complete = strconv.Itoa(*record.PercentComplete) + "%"
		}
		rows[i] = []render.Cell{
			render.Plain(record.Group), render.Plain(record.Name), render.Plain(words), render.Amount(record.TargetAmount, currency),
			render.Amount(record.Assigned, currency), render.Amount(record.Activity, currency), render.Available(record.Available, currency, colors),
			underfunded, render.Plain(complete), render.Plain(render.OptionalInt(record.MonthsToAssign)),
		}
	}
	return rows
}

// summary is the count line under the table, counting every target even
// when --underfunded kept only some rows.
func (r fundingReport) summary(month string, currency *ynab.CurrencyFormat) string {
	return fmt.Sprintf("%s in %s, %d underfunded, %s still needed", render.CountNoun(r.targets, "target", "targets"), month, r.underfundedCount, r.underfundedTotal.Format(currency))
}

func newReportsFunding(a *app) *cobra.Command {
	var output outputFlags
	var monthText string
	var underfundedOnly bool
	command := &cobra.Command{
		Use: "funding [--month M] [--underfunded]", Short: "List every target with what it still needs for a month",
		Long: `List one row per category with a target for one month: group, name,
target type in words, target amount, assigned, activity, available,
underfunded, percent complete, and months left in the target's period.
Rows keep the plan's order and cover the categories 'categories list'
shows, so hidden categories and the API's own are left out.
--underfunded keeps only the targets still needing money this month.
The summary line counts the underfunded targets and totals what they
need, whether or not --underfunded is set; 'ynab months fund
--all-underfunded' assigns that total.

Three requests: the plans endpoint, the month, whose rows carry the
amounts and the target fields, and the categories endpoint, which
supplies the group names and the hidden flags. For a "plan your
spending" target in a future month, the API's underfunded amount counts
funding from earlier periods, which the YNAB app ignores, so the two
can differ. --jsonl and --csv carry the target type as the API's code:
TB target balance, TBD target balance by date, MF monthly funding, NEED
plan your spending, DEBT debt payoff.

` + planHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab reports funding
  ynab reports funding --underfunded
  ynab reports funding --month 2026-10 --csv > october-targets.csv
  ynab reports funding --jsonl | jq 'select(.percent_complete < 100)'`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
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
			month, err := client.Month(cmd.Context(), plan.ID, apiMonth)
			if err != nil {
				return err
			}
			groups, err := client.Categories(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}

			report, err := newFundingReport(groups, month, underfundedOnly)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return render.JSONL(out, report.records)
			case output.csv:
				return render.CSV(out, report.records)
			}
			if err := render.Table(out, fundingColumns, report.rows(plan.CurrencyFormat, a.palette(cmd))); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, report.summary(dates.ShortMonth(apiMonth), plan.CurrencyFormat))
			return err
		}),
	}
	output.bind(command, true)
	bindMonthFlag(command, &monthText)
	command.Flags().BoolVar(&underfundedOnly, "underfunded", false, "Only targets still needing money")
	return command
}
