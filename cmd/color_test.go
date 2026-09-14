package cmd

import "testing"

func TestResolveColor(t *testing.T) {
	cases := []struct {
		name              string
		mode              colorMode
		explicit, noColor bool
		terminal, want    bool
	}{
		{name: "auto on terminal", mode: colorAuto, terminal: true, want: true},
		{name: "auto piped", mode: colorAuto, terminal: false, want: false},
		{name: "auto under NO_COLOR", mode: colorAuto, noColor: true, terminal: true, want: false},
		{name: "explicit auto ignores NO_COLOR", mode: colorAuto, explicit: true, noColor: true, terminal: true, want: true},
		{name: "always piped under NO_COLOR", mode: colorAlways, explicit: true, noColor: true, terminal: false, want: true},
		{name: "never on terminal", mode: colorNever, explicit: true, terminal: true, want: false},
	}
	for _, tc := range cases {
		if got := resolveColor(tc.mode, tc.explicit, tc.noColor, tc.terminal); got != tc.want {
			t.Errorf("%s: resolveColor() = %v, want %v", tc.name, got, tc.want)
		}
	}
}
