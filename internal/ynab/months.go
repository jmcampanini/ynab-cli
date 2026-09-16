package ynab

import (
	"context"
	"errors"
	"net/http"
)

// Month is one plan month's totals. Categories is filled only by Month,
// with each category's amounts for that month. AgeOfMoney is nil when the
// API has not computed it.
type Month struct {
	Activity      Amount     `json:"activity"`
	AgeOfMoney    *int       `json:"age_of_money"`
	Assigned      Amount     `json:"budgeted"`
	Categories    []Category `json:"categories"`
	Deleted       bool       `json:"deleted"`
	Income        Amount     `json:"income"`
	Month         string     `json:"month"`
	Note          *string    `json:"note"`
	ReadyToAssign Amount     `json:"to_be_budgeted"`
}

// Months returns every plan month's totals in the API's order, oldest
// first, without category rows.
func (c *Client) Months(ctx context.Context, planID string) ([]Month, error) {
	var data struct {
		Months []Month `json:"months"`
	}
	if err := c.get(ctx, "/plans/"+planID+"/months", &data); err != nil {
		return nil, err
	}
	return data.Months, nil
}

// Month returns one plan month with its category rows. month must be the
// API's YYYY-MM-01 form.
func (c *Client) Month(ctx context.Context, planID, month string) (Month, error) {
	var data struct {
		Month Month `json:"month"`
	}
	if err := c.get(ctx, "/plans/"+planID+"/months/"+month, &data); err != nil {
		return Month{}, err
	}
	return data.Month, nil
}

// UpdateMonthCategory sets a category's assigned amount for one month,
// the only month field the API writes, and returns the category with
// that month's amounts. month must be the API's YYYY-MM-01 form.
func (c *Client) UpdateMonthCategory(ctx context.Context, planID, month, categoryID string, assigned Amount) (Category, error) {
	var data struct {
		Category *Category `json:"category"`
	}
	body := struct {
		Category struct {
			Milliunits int64 `json:"budgeted"`
		} `json:"category"`
	}{}
	body.Category.Milliunits = int64(assigned)
	if err := c.do(ctx, http.MethodPatch, "/plans/"+planID+"/months/"+month+"/categories/"+categoryID, nil, body, &data); err != nil {
		return Category{}, err
	}
	if data.Category == nil {
		return Category{}, errors.New("the API updated the category but returned no record")
	}
	return *data.Category, nil
}
