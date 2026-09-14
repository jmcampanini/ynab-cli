package ynab

import (
	"strconv"
	"strings"
)

// Amount is a money value in milliunits, the API's integer unit of one
// thousandth of a currency unit. Outflows are negative.
type Amount int64

// String renders the amount as a plain decimal with two fractional digits
// and a leading minus for negatives, such as "-97.81".
func (a Amount) String() string {
	return a.decimal(2, ".", "")
}

// MarshalJSON writes the amount as a JSON number with two fractional digits.
func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(a.String()), nil
}

// Format renders the amount with a plan's currency settings, for example
// "$1,234.56" or "-97,81 €". A nil format falls back to String.
func (a Amount) Format(format *CurrencyFormat) string {
	if format == nil {
		return a.String()
	}

	digits := a.decimal(format.DecimalDigits, format.DecimalSeparator, format.GroupSeparator)
	if !format.DisplaySymbol || format.CurrencySymbol == "" {
		return digits
	}

	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign, digits = "-", digits[1:]
	}
	if format.SymbolFirst {
		return sign + format.CurrencySymbol + digits
	}
	return sign + digits + " " + format.CurrencySymbol
}

// decimal scales the milliunits to the requested number of fractional
// digits, rounding half away from zero, and inserts separators. Every
// arithmetic step stays in integers.
func (a Amount) decimal(fractionDigits int, decimalSeparator, groupSeparator string) string {
	if fractionDigits < 0 || fractionDigits > 3 {
		fractionDigits = 2
	}
	negative := a < 0
	milliunits := int64(a)
	if negative {
		milliunits = -milliunits
	}

	divisor := int64(1)
	for range 3 - fractionDigits {
		divisor *= 10
	}
	scaled := (milliunits + divisor/2) / divisor
	scale := int64(1)
	for range fractionDigits {
		scale *= 10
	}
	whole, fraction := scaled/scale, scaled%scale

	text := groupThousands(strconv.FormatInt(whole, 10), groupSeparator)
	if fractionDigits > 0 {
		fractionText := strconv.FormatInt(fraction, 10)
		text += decimalSeparator + strings.Repeat("0", fractionDigits-len(fractionText)) + fractionText
	}
	if negative {
		text = "-" + text
	}
	return text
}

func groupThousands(digits, separator string) string {
	if separator == "" || len(digits) <= 3 {
		return digits
	}
	var grouped strings.Builder
	head := len(digits) % 3
	if head > 0 {
		grouped.WriteString(digits[:head])
	}
	for i := head; i < len(digits); i += 3 {
		if grouped.Len() > 0 {
			grouped.WriteString(separator)
		}
		grouped.WriteString(digits[i : i+3])
	}
	return grouped.String()
}
