package cli

import (
	"github.com/spf13/cobra"
)

// bindMonthFlag adds --month with the shared default.
func bindMonthFlag(cmd *cobra.Command, text *string) {
	cmd.Flags().StringVar(text, "month", "current", "current, YYYY-MM, or YYYY-MM-01")
}
