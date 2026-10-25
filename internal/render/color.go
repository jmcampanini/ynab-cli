package render

// Palette paints human table cells. A disabled Palette returns text as is,
// so padding computed on plain text stays correct.
type Palette struct {
	Enabled bool
}

// Red paints nonempty text red when the palette is enabled.
func (p Palette) Red(text string) string { return p.wrap("31", text) }

// Faint dims nonempty text when the palette is enabled.
func (p Palette) Faint(text string) string { return p.wrap("2", text) }

func (p Palette) wrap(code, text string) string {
	if !p.Enabled || text == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

// ResolveColor applies the explicit mode before environment and terminal defaults.
func ResolveColor(mode string, explicit, noColor, terminal bool) bool {
	switch {
	case explicit && mode == "always":
		return true
	case explicit && mode == "never":
		return false
	case explicit:
		return terminal
	default:
		return terminal && !noColor
	}
}
