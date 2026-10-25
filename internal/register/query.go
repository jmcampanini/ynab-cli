// Package register selects transaction endpoints and their server-side filters.
package register

import (
	"context"
	"time"

	"github.com/jmcampanini/ynab-cli/internal/dates"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// Query contains resolved inputs for selecting a transaction endpoint.
type Query struct {
	AccountID     string
	Month         string
	Since         string
	Until         string
	Unapproved    bool
	Uncategorized bool
}

// Kind names the endpoint a filter set is served from.
type Kind string

// Transaction endpoint kinds, ordered by the selection rules in Query.Listing.
const (
	Account Kind = "account"
	Month   Kind = "month"
	Plan    Kind = "plan"
)

// Listing is the endpoint choice: the most specific of the account, month,
// and plan listings, plus the query the API applies. When the account
// endpoint wins, --month narrows the dates instead.
type Listing struct {
	Kind   Kind
	Filter ynab.TransactionFilter
}

// Listing picks the endpoint and the API-side query. The API accepts one
// type; uncategorized narrows the candidate rows at the API, and
// unapproved stays client-side when both are set.
func (f Query) Listing() Listing {
	query := ynab.TransactionFilter{SinceDate: f.Since, UntilDate: f.Until}
	switch {
	case f.Uncategorized:
		query.Type = ynab.TypeUncategorized
	case f.Unapproved:
		query.Type = ynab.TypeUnapproved
	}
	switch {
	case f.AccountID != "" && f.Month != "":
		start, end := monthBounds(f.Month)
		if query.SinceDate == "" || query.SinceDate < start {
			query.SinceDate = start
		}
		if query.UntilDate == "" || query.UntilDate > end {
			query.UntilDate = end
		}
		return Listing{Kind: Account, Filter: query}
	case f.AccountID != "":
		return Listing{Kind: Account, Filter: query}
	case f.Month != "":
		return Listing{Kind: Month, Filter: query}
	}
	return Listing{Kind: Plan, Filter: query}
}

// monthBounds returns the first and last dates of an API month.
func monthBounds(apiMonth string) (string, string) {
	start, err := time.Parse(dates.APIMonth, apiMonth)
	if err != nil {
		return apiMonth, apiMonth
	}
	return start.Format(time.DateOnly), start.AddDate(0, 1, -1).Format(time.DateOnly)
}

// Fetch performs the chosen listing.
func (l Listing) Fetch(ctx context.Context, client *ynab.Client, planID string, f Query) ([]ynab.Transaction, error) {
	switch l.Kind {
	case Account:
		return client.AccountTransactions(ctx, planID, f.AccountID, l.Filter)
	case Month:
		return client.MonthTransactions(ctx, planID, f.Month, l.Filter)
	}
	return client.Transactions(ctx, planID, l.Filter)
}
