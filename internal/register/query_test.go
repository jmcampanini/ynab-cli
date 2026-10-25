package register

import (
	"testing"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

func TestTransactionFiltersChooseTheListing(t *testing.T) {
	cases := []struct {
		name    string
		filters Query
		want    Listing
	}{
		{"plan", Query{}, Listing{Kind: Plan}},
		{"dates", Query{Since: "2026-08-01", Until: "2026-08-31"}, Listing{Kind: Plan, Filter: ynab.TransactionFilter{SinceDate: "2026-08-01", UntilDate: "2026-08-31"}}},
		{"account", Query{AccountID: "a1"}, Listing{Kind: Account}},
		{"month", Query{Month: "2026-08-01"}, Listing{Kind: Month}},
		{"account over month", Query{AccountID: "a1", Month: "2026-08-01"}, Listing{Kind: Account, Filter: ynab.TransactionFilter{SinceDate: "2026-08-01", UntilDate: "2026-08-31"}}},
		{"account, month, and narrower dates", Query{AccountID: "a1", Month: "2026-08-01", Since: "2026-08-10", Until: "2026-09-30"}, Listing{Kind: Account, Filter: ynab.TransactionFilter{SinceDate: "2026-08-10", UntilDate: "2026-08-31"}}},
		{"unapproved", Query{Unapproved: true}, Listing{Kind: Plan, Filter: ynab.TransactionFilter{Type: "unapproved"}}},
		{"uncategorized wins", Query{Unapproved: true, Uncategorized: true, Month: "2026-09-01"}, Listing{Kind: Month, Filter: ynab.TransactionFilter{Type: "uncategorized"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.filters.Listing()

			if got != tc.want {
				t.Errorf("Listing() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
