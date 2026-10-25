package cli

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
		Month:         dates.ShortMonth(month.Month),
		ReadyToAssign: month.ReadyToAssign,
	}
	if month.Note != nil {
		record.Note = *month.Note
	}
	return record
}

func newMonths(a *app) *cobra.Command {
	command := &cobra.Command{
		Use:   "months",
		Short: "Read the plan's months and move its money",
		Long: `Read the months of the configured plan: income, assigned, activity, ready
to assign, and age of money, plus each category's row for one month. A
bare 'ynab months' prints this help and exits 0. The writes set assigned
amounts: 'assign' sets one category, 'move' shifts money between two or
from ready to assign, 'cover' clears an overspent category's shortfall,
and 'fund' assigns what targets still need. The API cannot set month
notes or record a money movement; a move is two assigned-amount writes.

` + writeHelp + "\n\n" + planHelp + "\n\n" + monthHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(newMonthsList(a), newMonthsGet(a), newMonthsAssign(a), newMonthsMove(a), newMonthsCover(a), newMonthsFund(a))
	return command
}
