package cmd

import (
	"time"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// planRecord is the --jsonl and --csv shape of a plan.
type planRecord struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	FirstMonth     string    `json:"first_month"`
	LastMonth      string    `json:"last_month"`
	LastModifiedOn time.Time `json:"last_modified_on"`
	Currency       string    `json:"currency,omitempty"`
	DateFormat     string    `json:"date_format,omitempty"`
	Configured     bool      `json:"configured"`
}

func newPlanRecord(plan ynab.Plan, configured bool) planRecord {
	record := planRecord{
		Configured:     configured,
		FirstMonth:     shortMonth(plan.FirstMonth),
		ID:             plan.ID,
		LastModifiedOn: plan.LastModifiedOn,
		LastMonth:      shortMonth(plan.LastMonth),
		Name:           plan.Name,
	}
	if plan.CurrencyFormat != nil {
		record.Currency = plan.CurrencyFormat.ISOCode
	}
	if plan.DateFormat != nil {
		record.DateFormat = plan.DateFormat.Format
	}
	return record
}

func newPlans(a *app) *cobra.Command {
	command := &cobra.Command{
		Use: "plans", Short: "List plans and read the configured plan",
		Long: `Read the plans the token can access. A bare 'ynab plans' prints this
help and exits 0. 'plans list' needs only a token; 'plans get' and
'plans status' also need the configured plan.

` + planHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(newPlansList(a), newPlansGet(a), newPlansStatus(a))
	return command
}
