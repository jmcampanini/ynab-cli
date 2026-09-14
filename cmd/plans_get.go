package cmd

import (
	"time"

	"github.com/spf13/cobra"
)

func newPlansGet(a *app) *cobra.Command {
	var output outputFlags
	command := &cobra.Command{
		Use: "get", Short: "Show the configured plan's summary and settings",
		Long: `Show the configured plan: name, ID, first and last month, last
modification, and the currency and date format that human tables use.
One request to the plans endpoint. --jsonl prints one object.

` + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab plans get
  ynab plans get --plan Household --jsonl`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			loaded, client, err := a.connect(cmd)
			if err != nil {
				return err
			}
			plan, err := selectedPlan(cmd.Context(), client, loaded.Config)
			if err != nil {
				return err
			}

			record := newPlanRecord(plan, true)
			out := cmd.OutOrStdout()
			if output.jsonl {
				return writeJSONL(out, []planRecord{record})
			}
			currency, dateFormat := "unavailable", "unavailable"
			if plan.CurrencyFormat != nil {
				currency = plan.CurrencyFormat.ISOCode + " (" + plan.CurrencyFormat.ExampleFormat + ")"
			}
			if plan.DateFormat != nil {
				dateFormat = plan.DateFormat.Format
			}
			return writeFields(out, [][2]string{
				{"name", record.Name},
				{"id", record.ID},
				{"first month", record.FirstMonth},
				{"last month", record.LastMonth},
				{"last modified", record.LastModifiedOn.UTC().Format(time.DateTime)},
				{"currency", currency},
				{"date format", dateFormat},
			})
		}),
	}
	output.bind(command, false)
	return command
}
