package ynab

import (
	"context"
	"time"
)

// CurrencyFormat is a plan's currency display settings.
type CurrencyFormat struct {
	CurrencySymbol   string `json:"currency_symbol"`
	DecimalDigits    int    `json:"decimal_digits"`
	DecimalSeparator string `json:"decimal_separator"`
	DisplaySymbol    bool   `json:"display_symbol"`
	ExampleFormat    string `json:"example_format"`
	GroupSeparator   string `json:"group_separator"`
	ISOCode          string `json:"iso_code"`
	SymbolFirst      bool   `json:"symbol_first"`
}

// DateFormat is a plan's date display setting, such as "MM/DD/YYYY".
type DateFormat struct {
	Format string `json:"format"`
}

// Plan is the summary record the plans endpoint returns. The API leaves
// CurrencyFormat and DateFormat null for some plans.
type Plan struct {
	CurrencyFormat *CurrencyFormat `json:"currency_format"`
	DateFormat     *DateFormat     `json:"date_format"`
	FirstMonth     string          `json:"first_month"`
	ID             string          `json:"id"`
	LastModifiedOn time.Time       `json:"last_modified_on"`
	LastMonth      string          `json:"last_month"`
	Name           string          `json:"name"`
}

// Plans returns every plan the token can read.
func (c *Client) Plans(ctx context.Context) ([]Plan, error) {
	var data struct {
		Plans []Plan `json:"plans"`
	}
	if err := c.get(ctx, "/plans", &data); err != nil {
		return nil, err
	}
	return data.Plans, nil
}
