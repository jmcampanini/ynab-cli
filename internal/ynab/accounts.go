package ynab

import (
	"context"
	"time"
)

// Account is one plan account. Balances are milliunits.
type Account struct {
	Balance             Amount     `json:"balance"`
	ClearedBalance      Amount     `json:"cleared_balance"`
	Closed              bool       `json:"closed"`
	Deleted             bool       `json:"deleted"`
	DirectImportInError bool       `json:"direct_import_in_error"`
	DirectImportLinked  bool       `json:"direct_import_linked"`
	ID                  string     `json:"id"`
	LastReconciledAt    *time.Time `json:"last_reconciled_at"`
	Name                string     `json:"name"`
	Note                *string    `json:"note"`
	OnPlan              bool       `json:"on_budget"`
	TransferPayeeID     *string    `json:"transfer_payee_id"`
	Type                string     `json:"type"`
	UnclearedBalance    Amount     `json:"uncleared_balance"`
}

// Accounts returns every account in the plan, including closed ones.
// Deleted accounts only appear in delta requests, which the CLI never makes.
func (c *Client) Accounts(ctx context.Context, planID string) ([]Account, error) {
	var data struct {
		Accounts []Account `json:"accounts"`
	}
	if err := c.get(ctx, "/plans/"+planID+"/accounts", &data); err != nil {
		return nil, err
	}
	return data.Accounts, nil
}
