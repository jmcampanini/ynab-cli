// Package input parses command operands and flag values.
package input

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/jmcampanini/ynab-cli/internal/dates"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// OptionalDate parses a date flag, blank when the flag is unset.
func OptionalDate(text string, now time.Time) (string, error) {
	if text == "" {
		return "", nil
	}
	return dates.Date(text, now)
}

// Amount parses an amount typed in currency units at the plan's decimal
// precision.
func Amount(text string, plan ynab.Plan) (ynab.Amount, error) {
	return ynab.ParseAmount(text, ynab.DecimalDigits(plan))
}

// NameLength rejects an empty name or one past the API's limit, counted
// in characters.
func NameLength(name string, limit int) error {
	if name == "" {
		return errors.New("NAME must not be empty")
	}
	if count := utf8.RuneCountInString(name); count > limit {
		return fmt.Errorf("NAME is %d characters; the API allows at most %d", count, limit)
	}
	return nil
}
