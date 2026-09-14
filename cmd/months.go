package cmd

import (
	"github.com/jmcampanini/ynab-cli/internal/dates"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// monthRecord is the --jsonl and --csv shape of a plan month's totals.
type monthRecord struct {
	Month         string      `json:"month"`
	Note          string      `json:"note,omitempty"`
	Income        ynab.Amount `json:"income"`
	Assigned      ynab.Amount `json:"assigned"`
	Activity      ynab.Amount `json:"activity"`
	ReadyToAssign ynab.Amount `json:"ready_to_assign"`
	AgeOfMoney    *int        `json:"age_of_money,omitempty"`
}

func newMonthRecord(month ynab.Month) monthRecord {
	record := monthRecord{
		Activity:      month.Activity,
		AgeOfMoney:    month.AgeOfMoney,
		Assigned:      month.Assigned,
		Income:        month.Income,
		Month:         shortMonth(month.Month),
		ReadyToAssign: month.ReadyToAssign,
	}
	if month.Note != nil {
		record.Note = *month.Note
	}
	return record
}

// shortMonth shortens the API's first-of-month date to YYYY-MM.
func shortMonth(date string) string {
	if len(date) >= 7 {
		return date[:7]
	}
	return date
}

// month parses a month operand against the injected clock. A bad form is
// a usage error.
func (a *app) month(text string) (string, error) {
	apiMonth, err := dates.Month(text, a.deps.now())
	if err != nil {
		return "", usageError(err.Error())
	}
	return apiMonth, nil
}

// bindMonthFlag adds --month with the shared default.
func bindMonthFlag(cmd *cobra.Command, text *string) {
	cmd.Flags().StringVar(text, "month", "current", "Month: current, YYYY-MM, YYYY-MM-01")
}

func newMonths(a *app) *cobra.Command {
	command := &cobra.Command{
		Use: "months", Short: "List and read the plan's months",
		Long: `Read the months of the configured plan: income, assigned, activity, ready
to assign, and age of money, plus each category's row for one month. A
bare 'ynab months' prints this help and exits 0. Money moves arrive in a
later release; the API cannot set month notes.

` + planHelp + "\n\n" + monthHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(newMonthsList(a), newMonthsGet(a))
	return command
}
