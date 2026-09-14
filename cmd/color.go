package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type colorMode string

const (
	colorAuto   colorMode = "auto"
	colorAlways colorMode = "always"
	colorNever  colorMode = "never"
)

// colorValue binds --color to a colorMode and rejects other values.
type colorValue struct {
	mode *colorMode
}

func (v colorValue) Set(value string) error {
	switch mode := colorMode(value); mode {
	case colorAuto, colorAlways, colorNever:
		*v.mode = mode
		return nil
	default:
		return fmt.Errorf("invalid color mode %q: must be auto, always, or never", value)
	}
}

func (v colorValue) String() string { return string(*v.mode) }

func (colorValue) Type() string { return "mode" }

// palette paints human table cells. A disabled palette returns text as is,
// so padding computed on plain text stays correct.
type palette struct {
	enabled bool
}

func (p palette) red(text string) string { return p.wrap("31", text) }

func (p palette) faint(text string) string { return p.wrap("2", text) }

func (p palette) wrap(code, text string) string {
	if !p.enabled || text == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

// palette resolves color for the command's stdout: an explicit --color
// value wins, then NO_COLOR, then terminal detection with TERM=dumb off.
func (a *app) palette(cmd *cobra.Command) palette {
	explicit := cmd.Root().PersistentFlags().Changed("color")
	terminal := a.deps.isTerminal(cmd.OutOrStdout()) && a.deps.lookupEnv("TERM") != "dumb"
	noColor := a.deps.lookupEnv("NO_COLOR") != ""
	return palette{enabled: resolveColor(a.color, explicit, noColor, terminal)}
}

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
