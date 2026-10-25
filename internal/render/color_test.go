package render

import "testing"

func TestResolveColor(t *testing.T) {
	cases := []struct {
		name              string
		mode              string
		explicit, noColor bool
		terminal, want    bool
	}{
		{name: "auto on terminal", mode: "auto", terminal: true, want: true},
		{name: "auto piped", mode: "auto", terminal: false, want: false},
		{name: "auto under NO_COLOR", mode: "auto", noColor: true, terminal: true, want: false},
		{name: "explicit auto ignores NO_COLOR", mode: "auto", explicit: true, noColor: true, terminal: true, want: true},
		{name: "always piped under NO_COLOR", mode: "always", explicit: true, noColor: true, terminal: false, want: true},
		{name: "never on terminal", mode: "never", explicit: true, terminal: true, want: false},
	}
	for _, tc := range cases {
		if got := ResolveColor(tc.mode, tc.explicit, tc.noColor, tc.terminal); got != tc.want {
			t.Errorf("%s: resolveColor() = %v, want %v", tc.name, got, tc.want)
		}
	}
}
