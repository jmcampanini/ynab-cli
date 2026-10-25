package dates

import (
	"time"

	"github.com/jmcampanini/ynab-cli/internal/status"
)

// MonthInput parses a month operand against the injected clock. A bad form is
// a usage error.
func MonthInput(text string, now time.Time) (string, error) {
	apiMonth, err := Month(text, now)
	if err != nil {
		return "", status.Usage(err.Error())
	}
	return apiMonth, nil
}

// DateInput parses a date operand against the injected clock. A bad form is a
// usage error.
func DateInput(text string, now time.Time) (string, error) {
	date, err := Date(text, now)
	if err != nil {
		return "", status.Usage(err.Error())
	}
	return date, nil
}

// OptionalDate parses a date flag, blank when the flag is unset.
func OptionalDate(text string, now time.Time) (string, error) {
	if text == "" {
		return "", nil
	}
	return DateInput(text, now)
}
