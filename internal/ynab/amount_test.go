package ynab

import (
	"encoding/json"
	"testing"
)

func TestAmountString(t *testing.T) {
	cases := map[Amount]string{0: "0.00", -97810: "-97.81", 450000: "450.00", 1234567: "1234.57", -5: "-0.01", 4: "0.00"}
	for amount, want := range cases {
		if got := amount.String(); got != want {
			t.Errorf("Amount(%d).String() = %q, want %q", amount, got, want)
		}
	}
}

func TestAmountMarshalsAsJSONNumber(t *testing.T) {
	got, err := json.Marshal(struct{ Balance Amount }{Balance: -97810})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"Balance":-97.81}` {
		t.Errorf("json = %s, want a bare number", got)
	}
}

func TestAmountFormat(t *testing.T) {
	usd := &CurrencyFormat{CurrencySymbol: "$", DecimalDigits: 2, DecimalSeparator: ".", DisplaySymbol: true, GroupSeparator: ",", SymbolFirst: true}
	eur := &CurrencyFormat{CurrencySymbol: "€", DecimalDigits: 2, DecimalSeparator: ",", DisplaySymbol: true, GroupSeparator: ".", SymbolFirst: false}
	jpy := &CurrencyFormat{CurrencySymbol: "¥", DecimalDigits: 0, DecimalSeparator: ".", DisplaySymbol: true, GroupSeparator: ",", SymbolFirst: true}
	cases := []struct {
		amount Amount
		format *CurrencyFormat
		want   string
	}{
		{1234560, usd, "$1,234.56"},
		{-97810, usd, "-$97.81"},
		{0, usd, "$0.00"},
		{1234567890, usd, "$1,234,567.89"},
		{-1234560, eur, "-1.234,56 €"},
		{1234500, jpy, "¥1,235"},
		{-97810, nil, "-97.81"},
		{500, &CurrencyFormat{DecimalDigits: 2, DecimalSeparator: ".", DisplaySymbol: false}, "0.50"},
	}
	for _, tc := range cases {
		if got := tc.amount.Format(tc.format); got != tc.want {
			t.Errorf("Amount(%d).Format(%+v) = %q, want %q", tc.amount, tc.format, got, tc.want)
		}
	}
}

func TestParseAmount(t *testing.T) {
	cases := []struct {
		text   string
		digits int
		want   Amount
		ok     bool
	}{
		{"450", 2, 450000, true},
		{"-97.81", 2, -97810, true},
		{"+12.5", 2, 12500, true},
		{"0.5", 2, 500, true},
		{"0", 2, 0, true},
		{"-0.001", 3, -1, true},
		{"1.234", 2, 0, false},
		{"1,000", 2, 0, false},
		{"$5", 2, 0, false},
		{"", 2, 0, false},
		{"-", 2, 0, false},
		{".5", 2, 0, false},
		{"5.", 2, 0, false},
		{"1e3", 2, 0, false},
		{"12.34", 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			got, err := ParseAmount(tc.text, tc.digits)

			if (err == nil) != tc.ok || got != tc.want {
				t.Errorf("ParseAmount(%q, %d) = %d, %v, want %d, ok=%v", tc.text, tc.digits, got, err, tc.want, tc.ok)
			}
		})
	}
}
