package cli

import (
	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/spf13/cobra"
)

// palette resolves color for the command's stdout: an explicit --color
// value wins, then NO_COLOR, then terminal detection with TERM=dumb off.
func (a *app) palette(cmd *cobra.Command) render.Palette {
	explicit := cmd.Root().PersistentFlags().Changed("color")
	terminal := a.deps.isTerminal(cmd.OutOrStdout()) && a.deps.lookupEnv("TERM") != "dumb"
	noColor := a.deps.lookupEnv("NO_COLOR") != ""
	return render.Palette{Enabled: resolveColor(a.color, explicit, noColor, terminal)}
}

// resolveColor applies the explicit mode before environment and terminal defaults.
func resolveColor(mode colorMode, explicit, noColor, terminal bool) bool {
	switch {
	case explicit && mode == colorAlways:
		return true
	case explicit && mode == colorNever:
		return false
	case explicit:
		return terminal
	default:
		return terminal && !noColor
	}
}
