package cli

import "testing"

func TestResolveColor(t *testing.T) {
	cases := []struct {
		name              string
		mode              colorMode
		noColor, terminal bool
		dumb, want        bool
	}{
		{name: "auto on terminal", mode: colorAuto, terminal: true, want: true},
		{name: "auto piped", mode: colorAuto, terminal: false, want: false},
		{name: "auto on dumb terminal", mode: colorAuto, terminal: true, dumb: true, want: false},
		{name: "auto under NO_COLOR", mode: colorAuto, noColor: true, terminal: true, want: false},
		{name: "always piped under NO_COLOR", mode: colorAlways, noColor: true, terminal: false, want: true},
		{name: "always on dumb terminal", mode: colorAlways, terminal: true, dumb: true, want: true},
		{name: "never on terminal", mode: colorNever, terminal: true, want: false},
	}
	for _, tc := range cases {
		if got := resolveColor(tc.mode, tc.noColor, tc.terminal, tc.dumb); got != tc.want {
			t.Errorf("%s: resolveColor() = %v, want %v", tc.name, got, tc.want)
		}
	}
}
