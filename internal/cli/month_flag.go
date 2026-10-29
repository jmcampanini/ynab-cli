package cli

import (
	"github.com/spf13/cobra"
)

// bindMonthFlag adds --month; configureDefaults resolves its default at run time.
func bindMonthFlag(cmd *cobra.Command, text *string) {
	cmd.Flags().StringVar(text, "month", "", "Month: current or YYYY-MM[-01]")
	useCommandDefault(cmd.Flags().Lookup("month"))
}
