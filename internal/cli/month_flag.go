package cli

import (
	"github.com/spf13/cobra"
)

// bindMonthFlag adds --month with the shared default.
func bindMonthFlag(cmd *cobra.Command, text *string) {
	cmd.Flags().StringVar(text, "month", "current", "Month: current, YYYY-MM, YYYY-MM-01")
}
