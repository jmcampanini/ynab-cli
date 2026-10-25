package render

import (
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// HiddenSuffix marks hidden names in human output.
func HiddenSuffix(name string, hidden bool) string {
	if hidden {
		return name + " (hidden)"
	}
	return name
}

// Amount formats currency as an unpainted table cell.
func Amount(amount ynab.Amount, currency *ynab.CurrencyFormat) Cell {
	return Plain(amount.Format(currency))
}

// Available paints an overspent amount red.
func Available(amount ynab.Amount, currency *ynab.CurrencyFormat, colors Palette) Cell {
	c := Amount(amount, currency)
	if amount < 0 {
		c.Paint = colors.Red
	}
	return c
}

// PaintRow makes every cell of a hidden row faint.
func PaintRow(row []Cell, hidden bool, colors Palette) []Cell {
	if hidden {
		for i := range row {
			row[i].Paint = colors.Faint
		}
	}
	return row
}
