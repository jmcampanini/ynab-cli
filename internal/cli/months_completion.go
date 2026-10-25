package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

// completeSources offers the categories plus the ready-to-assign operand
// of --from and --to.
func (a *app) completeSources() completer {
	categories := a.completeCategories()
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		names, directive := categories(cmd, args, toComplete)
		if strings.HasPrefix(readyToAssignOperand, strings.ToLower(toComplete)) {
			names = append(names, readyToAssignOperand)
		}
		return names, directive
	}
}
