package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newPlansList(a *app) *cobra.Command {
	var output outputFlags
	command := &cobra.Command{
		Use: "list", Short: "List every plan the token can read",
		Long: `List every plan the token can read with its ID, name, last modification,
and first and last month. The plan named by the configuration, if any, is
marked with * in the table and "configured": true in JSONL and CSV. This
command needs a token but no configured plan, so it is the way to find the
value for plan. One request to the plans endpoint.

` + configHelp + "\n\n" + outputHelp,
		Example: `  ynab plans list
  ynab plans list --jsonl | jq -r .id`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			loaded, client, err := a.connect(cmd)
			if err != nil {
				return err
			}
			plans, err := client.Plans(cmd.Context())
			if err != nil {
				return err
			}

			configured := map[string]bool{}
			for _, plan := range matchPlan(loaded.Config.Plan, plans) {
				configured[plan.ID] = true
			}
			records := make([]planRecord, len(plans))
			for i, plan := range plans {
				records[i] = newPlanRecord(plan, configured[plan.ID])
			}

			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return writeJSONL(out, records)
			case output.csv:
				return writeCSV(out, records)
			}
			columns := []column{{name: " "}, {name: "NAME"}, {name: "ID"}, {name: "LAST MODIFIED"}, {name: "FIRST MONTH"}, {name: "LAST MONTH"}}
			rows := make([][]cell, len(records))
			for i, record := range records {
				marker := ""
				if record.Configured {
					marker = "*"
				}
				rows[i] = []cell{plain(marker), plain(record.Name), plain(record.ID), plain(record.LastModifiedOn.UTC().Format(time.DateTime)), plain(record.FirstMonth), plain(record.LastMonth)}
			}
			if err := writeTable(out, columns, rows); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, countNoun(len(records), "plan", "plans"))
			return err
		}),
	}
	output.bind(command, true)
	return command
}
