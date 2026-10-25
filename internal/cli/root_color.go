package cli

import (
	"fmt"
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
