package cli

import (
	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/spf13/cobra"
)

// palette resolves color for the command's stdout. always and never
// decide alone; auto, typed or defaulted, needs a terminal that is not
// TERM=dumb and an empty or unset NO_COLOR.
func (a *app) palette(cmd *cobra.Command) render.Palette {
	terminal := a.deps.isTerminal(cmd.OutOrStdout())
	dumb := a.deps.lookupEnv("TERM") == "dumb"
	noColor := a.deps.lookupEnv("NO_COLOR") != ""
	return render.Palette{Enabled: resolveColor(a.color, noColor, terminal, dumb)}
}

// resolveColor applies the mode to the detected inputs. Terminal status
// and the dumb profile stay separate: a stream can be a terminal whose
// TERM=dumb profile renders no color.
func resolveColor(mode colorMode, noColor, terminal, dumb bool) bool {
	switch mode {
	case colorAlways:
		return true
	case colorNever:
		return false
	default:
		return terminal && !dumb && !noColor
	}
}
