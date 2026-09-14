package dates

import (
	"testing"
	"time"
)

// now is late on the last evening of September in New York, which is
// already October in UTC.
var now = time.Date(2026, time.September, 30, 22, 0, 0, 0, time.FixedZone("EDT", -4*60*60))

func TestMonth(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{"current", "2026-10-01"},
		{"2026-08", "2026-08-01"},
		{"2026-08-01", "2026-08-01"},
		{"2026-8", ""},
		{"2026-08-15", ""},
		{"today", ""},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			got, err := Month(tc.text, now)

			if tc.want == "" {
				if err == nil {
					t.Fatalf("Month(%q) = %q, want an error", tc.text, got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Errorf("Month(%q) = %q, %v; want %q", tc.text, got, err, tc.want)
			}
		})
	}
}

func TestDate(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{"today", "2026-09-30"},
		{"yesterday", "2026-09-29"},
		{"2026-02-28", "2026-02-28"},
		{"2026-02-30", ""},
		{"2026-09", ""},
		{"current", ""},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			got, err := Date(tc.text, now)

			if tc.want == "" {
				if err == nil {
					t.Fatalf("Date(%q) = %q, want an error", tc.text, got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Errorf("Date(%q) = %q, %v; want %q", tc.text, got, err, tc.want)
			}
		})
	}
}
