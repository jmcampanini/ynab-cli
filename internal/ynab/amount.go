package ynab

import (
	"fmt"
	"math"
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

// Decimal renders the amount as a plain decimal with fractionDigits
// fractional digits and no separators or symbol, the form ParseAmount
// accepts back at that precision.
func (a Amount) Decimal(fractionDigits int) string {
	return a.decimal(fractionDigits, ".", "")
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

// ParseAmount parses an amount typed in currency units, such as "450",
// "-97.81", or "+12.5", into milliunits without floating point. It accepts
// an optional sign, digits, and at most fractionDigits digits after a
// period, and rejects group separators, currency symbols, and blanks.
// fractionDigits above 3 is treated as 3, the milliunit precision.
func ParseAmount(text string, fractionDigits int) (Amount, error) {
	fractionDigits = max(0, min(fractionDigits, 3))
	invalid := func() error {
		return fmt.Errorf("invalid amount %q: use digits with an optional sign and at most %d decimal places, without separators or symbols", text, fractionDigits)
	}

	rest := text
	negative := false
	if strings.HasPrefix(rest, "-") || strings.HasPrefix(rest, "+") {
		negative = rest[0] == '-'
		rest = rest[1:]
	}
	whole, fraction, hasPeriod := strings.Cut(rest, ".")
	if whole == "" || !allDigits(whole) {
		return 0, invalid()
	}
	if hasPeriod && (fraction == "" || len(fraction) > fractionDigits || !allDigits(fraction)) {
		return 0, invalid()
	}

	units, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, invalid()
	}
	var fractionValue int64
	if fraction != "" {
		if fractionValue, err = strconv.ParseInt(fraction+strings.Repeat("0", 3-len(fraction)), 10, 64); err != nil {
			return 0, invalid()
		}
	}
	if units > (math.MaxInt64-fractionValue)/1000 {
		return 0, invalid()
	}
	milliunits := units*1000 + fractionValue
	if negative {
		milliunits = -milliunits
	}
	return Amount(milliunits), nil
}

func allDigits(text string) bool {
	for _, r := range text {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
