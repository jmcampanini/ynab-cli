package ynab

import (
	"context"
	"time"
)

// Category is one plan line with the amounts of one month. The target
// fields mirror the API's goal fields and are nil when the API sends null;
// TargetType is nil when the category has no target, and TargetAmount is
// then 0 rather than null.
type Category struct {
	Activity              Amount     `json:"activity"`
	Assigned              Amount     `json:"budgeted"`
	Available             Amount     `json:"balance"`
	Deleted               bool       `json:"deleted"`
	GroupID               string     `json:"category_group_id"`
	GroupName             string     `json:"category_group_name"`
	Hidden                bool       `json:"hidden"`
	ID                    string     `json:"id"`
	Internal              bool       `json:"internal"`
	Name                  string     `json:"name"`
	Note                  *string    `json:"note"`
	TargetAmount          Amount     `json:"goal_target"`
	TargetCadence         *int       `json:"goal_cadence"`
	TargetCadenceFreq     *int       `json:"goal_cadence_frequency"`
	TargetCreationMonth   *string    `json:"goal_creation_month"`
	TargetDate            *string    `json:"goal_target_date"`
	TargetDay             *int       `json:"goal_day"`
	TargetMonthsToAssign  *int       `json:"goal_months_to_budget"`
	TargetNeedsWholeAmt   *bool      `json:"goal_needs_whole_amount"`
	TargetOverallFunded   *Amount    `json:"goal_overall_funded"`
	TargetOverallLeft     *Amount    `json:"goal_overall_left"`
	TargetPercentComplete *int       `json:"goal_percentage_complete"`
	TargetSnoozedAt       *time.Time `json:"goal_snoozed_at"`
	TargetType            *string    `json:"goal_type"`
	TargetUnderfunded     *Amount    `json:"goal_under_funded"`
}

// CategoryGroup is a grouping of categories. Categories carry the current
// month's amounts.
type CategoryGroup struct {
	Categories []Category `json:"categories"`
	Deleted    bool       `json:"deleted"`
	Hidden     bool       `json:"hidden"`
	ID         string     `json:"id"`
	Internal   bool       `json:"internal"`
	Name       string     `json:"name"`
}

// Categories returns every category group with its categories in the
// plan's display order, hidden ones included, with the current month's
// amounts. Deleted entities only appear in delta requests, which the CLI
// never makes.
func (c *Client) Categories(ctx context.Context, planID string) ([]CategoryGroup, error) {
	var data struct {
		Groups []CategoryGroup `json:"category_groups"`
	}
	if err := c.get(ctx, "/plans/"+planID+"/categories", &data); err != nil {
		return nil, err
	}
	return data.Groups, nil
}

// MonthCategory returns one category with the amounts of the given month,
// which must be the API's YYYY-MM-01 form.
func (c *Client) MonthCategory(ctx context.Context, planID, month, categoryID string) (Category, error) {
	var data struct {
		Category Category `json:"category"`
	}
	if err := c.get(ctx, "/plans/"+planID+"/months/"+month+"/categories/"+categoryID, &data); err != nil {
		return Category{}, err
	}
	return data.Category, nil
}
