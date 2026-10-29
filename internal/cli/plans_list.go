package cli

import (
	"fmt"
	"time"

	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/selection"
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
			for _, plan := range selection.MatchPlan(loaded.Config.Plan, plans) {
				configured[plan.ID] = true
			}
			records := make([]planRecord, len(plans))
			for i, plan := range plans {
				records[i] = newPlanRecord(plan, configured[plan.ID])
			}

			records = limitRecords(&output, records)
			out := cmd.OutOrStdout()
			if output.machine() {
				return writeRecords(out, output, records)
			}
			columns := []render.Column{{Name: " "}, {Name: "NAME"}, {Name: "ID"}, {Name: "LAST MODIFIED"}, {Name: "FIRST MONTH"}, {Name: "LAST MONTH"}}
			rows := make([][]render.Cell, len(records))
			for i, record := range records {
				marker := ""
				if record.Configured {
					marker = "*"
				}
				rows[i] = []render.Cell{render.Plain(marker), render.Plain(record.Name), render.Plain(record.ID), render.Plain(record.LastModifiedOn.UTC().Format(time.DateTime)), render.Plain(record.FirstMonth), render.Plain(record.LastMonth)}
			}
			if err := render.Table(out, columns, rows); err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, render.CountNoun(len(records), "plan", "plans"))
			return err
		}),
	}
	output.bindRead(command, true)
	return command
}
