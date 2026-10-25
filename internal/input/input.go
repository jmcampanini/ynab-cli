// Package input parses command operands and flag values, reporting a bad
// form as a usage error so nothing runs before it is fixed.
package input

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/jmcampanini/ynab-cli/internal/dates"
	"github.com/jmcampanini/ynab-cli/internal/status"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// Month parses a month operand against the injected clock.
func Month(text string, now time.Time) (string, error) {
	apiMonth, err := dates.Month(text, now)
	if err != nil {
		return "", status.Usage(err.Error())
	}
	return apiMonth, nil
}

// Date parses a date operand against the injected clock.
func Date(text string, now time.Time) (string, error) {
	date, err := dates.Date(text, now)
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
	return Date(text, now)
}

// Amount parses an amount typed in currency units at the plan's decimal
// precision.
func Amount(text string, plan ynab.Plan) (ynab.Amount, error) {
	amount, err := ynab.ParseAmount(text, ynab.DecimalDigits(plan))
	if err != nil {
		return 0, status.Usage(err.Error())
	}
	return amount, nil
}

// NameLength rejects an empty name or one past the API's limit, counted
// in characters.
func NameLength(name string, limit int) error {
	if name == "" {
		return status.Usage("NAME must not be empty")
	}
	if count := utf8.RuneCountInString(name); count > limit {
		return status.Usage(fmt.Sprintf("NAME is %d characters; the API allows at most %d", count, limit))
	}
	return nil
}
