// Package dates parses the month and date operands commands accept and
// returns the forms the YNAB API takes.
package dates

import (
	"fmt"
	"time"
)

// APIMonth is the layout of a month in API paths and responses.
const APIMonth = "2006-01-02"

// Month parses a month operand: "current", "YYYY-MM", or "YYYY-MM-01",
// and returns the API's YYYY-MM-01 form. "current" is the calendar month
// of now in UTC, which is how the API defines the current plan month.
func Month(text string, now time.Time) (string, error) {
	if text == "current" {
		year, month, _ := now.UTC().Date()
		return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format(APIMonth), nil
	}
	if parsed, err := time.Parse("2006-01", text); err == nil {
		return parsed.Format(APIMonth), nil
	}
	if parsed, err := time.Parse(APIMonth, text); err == nil && parsed.Day() == 1 {
		return parsed.Format(APIMonth), nil
	}
	return "", fmt.Errorf("invalid month %q: use current, YYYY-MM, or YYYY-MM-01", text)
}

// Date parses a date operand: "today", "yesterday", or "YYYY-MM-DD", and
// returns YYYY-MM-DD. The named days use now's own location, since a
// person enters transaction dates in local time.
func Date(text string, now time.Time) (string, error) {
	switch text {
	case "today":
		return now.Format(time.DateOnly), nil
	case "yesterday":
		return now.AddDate(0, 0, -1).Format(time.DateOnly), nil
	}
	if parsed, err := time.Parse(time.DateOnly, text); err == nil {
		return parsed.Format(time.DateOnly), nil
	}
	return "", fmt.Errorf("invalid date %q: use today, yesterday, or YYYY-MM-DD", text)
}
