package ynab

import "context"

// Payee is who a transaction paid. A transfer payee stands for an account
// and carries that account's ID in TransferAccountID.
type Payee struct {
	Deleted           bool    `json:"deleted"`
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	TransferAccountID *string `json:"transfer_account_id"`
}

// Payees returns every payee in the plan in the API's order. Deleted
// payees only appear in delta requests, which the CLI never makes.
func (c *Client) Payees(ctx context.Context, planID string) ([]Payee, error) {
	var data struct {
		Payees []Payee `json:"payees"`
	}
	if err := c.get(ctx, "/plans/"+planID+"/payees", &data); err != nil {
		return nil, err
	}
	return data.Payees, nil
}
