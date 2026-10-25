package ynab

import (
	"github.com/jmcampanini/ynab-cli/internal/status"
)

// DecimalDigits is the plan's currency precision, two when the plan has
// no currency format.
func DecimalDigits(plan Plan) int {
	if plan.CurrencyFormat != nil {
		return plan.CurrencyFormat.DecimalDigits
	}
	return 2
}

// InputAmount parses an amount typed in currency units at the plan's decimal
// precision. A bad value is a usage error.
func InputAmount(text string, plan Plan) (Amount, error) {
	amount, err := ParseAmount(text, DecimalDigits(plan))
	if err != nil {
		return 0, status.Usage(err.Error())
	}
	return amount, nil
}
