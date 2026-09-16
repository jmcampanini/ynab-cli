package ynab

import (
	"context"
	"errors"
	"net/http"
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

// SaveAccount is the body of an account create. Milliunits carries the
// opening balance as the API's integer. Type is one of the six the API
// creates: checking, savings, cash, creditCard, otherAsset, otherLiability.
type SaveAccount struct {
	Milliunits int64  `json:"balance"`
	Name       string `json:"name"`
	Type       string `json:"type"`
}

// CreateAccount creates an account and returns it as stored. The API
// cannot rename, close, or delete it afterwards.
func (c *Client) CreateAccount(ctx context.Context, planID string, account SaveAccount) (Account, error) {
	var data struct {
		Account *Account `json:"account"`
	}
	body := struct {
		Account SaveAccount `json:"account"`
	}{Account: account}
	if err := c.do(ctx, http.MethodPost, "/plans/"+planID+"/accounts", nil, body, &data); err != nil {
		return Account{}, err
	}
	if data.Account == nil {
		return Account{}, errors.New("the API created the account but returned no record")
	}
	return *data.Account, nil
}
