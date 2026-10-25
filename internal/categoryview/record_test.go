package categoryview

import "testing"

func TestTargetWords(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	cases := []struct {
		name   string
		target Target
		want   [2]string
	}{
		{"monthly", Target{Cadence: intPtr(1), CadenceFrequency: intPtr(1)}, [2]string{"monthly", "last day of the month"}},
		{"every other month", Target{Cadence: intPtr(1), CadenceFrequency: intPtr(2), Day: intPtr(15)}, [2]string{"every 2 months", "day 15 of the month"}},
		{"weekly on friday", Target{Cadence: intPtr(2), CadenceFrequency: intPtr(1), Day: intPtr(5)}, [2]string{"weekly", "Friday"}},
		{"weekly without a day", Target{Cadence: intPtr(2)}, [2]string{"weekly", ""}},
		{"every 3 months by code", Target{Cadence: intPtr(4)}, [2]string{"every 3 months", "last day of the month"}},
		{"yearly", Target{Cadence: intPtr(13), CadenceFrequency: intPtr(1)}, [2]string{"yearly", "last day of the month"}},
		{"every 2 years", Target{Cadence: intPtr(14)}, [2]string{"every 2 years", "last day of the month"}},
		{"none", Target{Cadence: intPtr(0)}, [2]string{"none", ""}},
		{"absent", Target{}, [2]string{"", ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := [2]string{tc.target.CadenceWords(), tc.target.DayWords()}

			if got != tc.want {
				t.Errorf("cadenceWords, dayWords = %q, want %q", got, tc.want)
			}
		})
	}
}
