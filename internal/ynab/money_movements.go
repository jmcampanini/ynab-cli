package ynab

import (
	"context"
	"time"
)

// MoneyMovement is a recorded move of assigned money between two
// categories in one month. A nil FromCategoryID or ToCategoryID means
// ready to assign. The API records movements; it cannot create them
// directly.
type MoneyMovement struct {
	Amount            Amount     `json:"amount"`
	FromCategoryID    *string    `json:"from_category_id"`
	GroupID           *string    `json:"money_movement_group_id"`
	ID                string     `json:"id"`
	Month             *string    `json:"month"`
	MovedAt           *time.Time `json:"moved_at"`
	Note              *string    `json:"note"`
	PerformedByUserID *string    `json:"performed_by_user_id"`
	ToCategoryID      *string    `json:"to_category_id"`
}

// MoneyMovements returns every money movement in the plan in the API's
// order.
func (c *Client) MoneyMovements(ctx context.Context, planID string) ([]MoneyMovement, error) {
	return c.listMoneyMovements(ctx, "/plans/"+planID+"/money_movements")
}

// MonthMoneyMovements returns one plan month's money movements. month must
// be the API's YYYY-MM-01 form.
func (c *Client) MonthMoneyMovements(ctx context.Context, planID, month string) ([]MoneyMovement, error) {
	return c.listMoneyMovements(ctx, "/plans/"+planID+"/months/"+month+"/money_movements")
}

func (c *Client) listMoneyMovements(ctx context.Context, path string) ([]MoneyMovement, error) {
	var data struct {
		MoneyMovements []MoneyMovement `json:"money_movements"`
	}
	if err := c.get(ctx, path, &data); err != nil {
		return nil, err
	}
	return data.MoneyMovements, nil
}
