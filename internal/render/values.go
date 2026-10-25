package render

import (
	"time"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// Text dereferences an optional API string, blank when null.
func Text(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// FlagWords joins a flag's color and custom name for human output.
func FlagWords(color, name string) string {
	if color == "" {
		return ""
	}
	if name == "" {
		return color
	}
	return color + " (" + name + ")"
}

// SplitColumns and SplitRow render a split's lines, of a transaction or a
// scheduled transaction, in a table under a single record.
var SplitColumns = []Column{{Name: "PAYEE"}, {Name: "CATEGORY"}, {Name: "MEMO"}, {Name: "AMOUNT", Right: true}}

// SplitRow formats one split line in the shared split table columns.
func SplitRow(payee, category, memo string, amount ynab.Amount, currency *ynab.CurrencyFormat) []Cell {
	return []Cell{Plain(payee), Plain(category), Plain(memo), Amount(amount, currency)}
}

// TransactionCount pluralizes a transaction count.
func TransactionCount(count int) string {
	return CountNoun(count, "transaction", "transactions")
}

// OptionalTime formats a timestamp in UTC, or returns blank for nil.
func OptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.DateTime)
}
