package cmd

import "github.com/spf13/cobra"

func newReports(a *app) *cobra.Command {
	command := &cobra.Command{
		Use: "reports", Short: "Derived reports over targets and spending",
		Long: `Read-only reports the API does not serve as such, derived from the
month, the categories, and the register of the configured plan. A bare
'ynab reports' prints this help and exits 0. 'funding' lists every target
with what it still needs for a month; 'spending' totals outflows and
inflows over a date range by category or payee. Both write --jsonl and
--csv in the shapes 'ynab help output-formats' describes.

` + planHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(newReportsFunding(a), newReportsSpending(a))
	return command
}
