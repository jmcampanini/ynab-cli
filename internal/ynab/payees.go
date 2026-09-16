package ynab

import (
	"context"
	"errors"
	"net/http"
)

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

// savedPayee is the data of the payee create and update responses.
type savedPayee struct {
	Payee *Payee `json:"payee"`
}

// CreatePayee creates a payee with the name and returns it as stored.
func (c *Client) CreatePayee(ctx context.Context, planID, name string) (Payee, error) {
	var data savedPayee
	if err := c.do(ctx, http.MethodPost, "/plans/"+planID+"/payees", nil, payeeBody(name), &data); err != nil {
		return Payee{}, err
	}
	if data.Payee == nil {
		return Payee{}, errors.New("the API created the payee but returned no record")
	}
	return *data.Payee, nil
}

// UpdatePayee renames a payee, the only field the API writes, and
// returns it as stored.
func (c *Client) UpdatePayee(ctx context.Context, planID, payeeID, name string) (Payee, error) {
	var data savedPayee
	if err := c.do(ctx, http.MethodPatch, "/plans/"+planID+"/payees/"+payeeID, nil, payeeBody(name), &data); err != nil {
		return Payee{}, err
	}
	if data.Payee == nil {
		return Payee{}, errors.New("the API updated the payee but returned no record")
	}
	return *data.Payee, nil
}

func payeeBody(name string) any {
	type payee struct {
		Name string `json:"name"`
	}
	return struct {
		Payee payee `json:"payee"`
	}{Payee: payee{Name: name}}
}
