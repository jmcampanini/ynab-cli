package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newCategoryGroupsList(a *app) *cobra.Command {
	var output outputFlags
	var includeHidden bool
	command := &cobra.Command{
		Use: "list", Short: "List the plan's category groups with their category counts",
		Long: `List the configured plan's category groups in the plan's order with each
group's ID and the number of categories 'categories list' would show in
it. Hidden groups are left out and hidden categories are not counted
unless --hidden is given, which shows hidden groups faint with a
"(hidden)" suffix and counts every category. Internal groups are shown.
Two requests: the plans endpoint, then the categories endpoint.

` + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab category-groups list
  ynab category-groups list --hidden --jsonl`,
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
			groups, err := client.Categories(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}

			var records []categoryGroupRecord
			hiddenOut := 0
			for _, group := range groups {
				if group.Hidden && !includeHidden {
					hiddenOut++
					continue
				}
				records = append(records, newCategoryGroupRecord(group, includeHidden))
			}

			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return writeJSONL(out, records)
			case output.csv:
				return writeCSV(out, records)
			}
			columns := []column{{name: "NAME"}, {name: "CATEGORIES", right: true}, {name: "ID"}}
			rows := make([][]cell, len(records))
			colors := a.palette(cmd)
			for i, record := range records {
				row := []cell{plain(hiddenSuffix(record.Name, record.Hidden)), plain(strconv.Itoa(record.CategoryCount)), plain(record.ID)}
				rows[i] = paintRow(row, record.Hidden, colors)
			}
			if err := writeTable(out, columns, rows); err != nil {
				return err
			}
			summary := countNoun(len(records), "group", "groups")
			if hiddenOut > 0 {
				summary += fmt.Sprintf(", %d hidden not shown", hiddenOut)
			}
			_, err = fmt.Fprintln(out, summary)
			return err
		}),
	}
	output.bind(command, true)
	command.Flags().BoolVar(&includeHidden, "hidden", false, "Include hidden groups and count hidden categories")
	return command
}
