package ynab

import (
	"context"
	"errors"
	"net/http"
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

// SaveCategory is the body of a category create or update. A zero field
// is left out of the body, so an update changes only the fields it names.
// TargetAmount carries milliunits as the API's integer; sending null
// removes the target. TargetFrequency makes a recurring NEED target and
// is monthly, weekly, or yearly.
type SaveCategory struct {
	GroupID             string          `json:"category_group_id,omitempty"`
	Name                string          `json:"name,omitempty"`
	Note                *string         `json:"note,omitempty"`
	TargetAmount        Nullable[int64] `json:"goal_target,omitzero"`
	TargetDate          *string         `json:"goal_target_date,omitempty"`
	TargetFrequency     string          `json:"goal_frequency,omitempty"`
	TargetNeedsWholeAmt *bool           `json:"goal_needs_whole_amount,omitempty"`
}

// SetTargetAmount records amount as the milliunits to send.
func (s *SaveCategory) SetTargetAmount(amount Amount) {
	s.TargetAmount = SomeValue(int64(amount))
}

// savedCategory is the data of the category create and update responses.
type savedCategory struct {
	Category *Category `json:"category"`
}

// CreateCategory creates a category in the group the body names and
// returns it as stored, with the current month's amounts.
func (c *Client) CreateCategory(ctx context.Context, planID string, category SaveCategory) (Category, error) {
	var data savedCategory
	body := struct {
		Category SaveCategory `json:"category"`
	}{Category: category}
	if err := c.do(ctx, http.MethodPost, "/plans/"+planID+"/categories", nil, body, &data); err != nil {
		return Category{}, err
	}
	if data.Category == nil {
		return Category{}, errors.New("the API created the category but returned no record")
	}
	return *data.Category, nil
}

// UpdateCategory changes the fields the body names and returns the
// category as stored, with the current month's amounts.
func (c *Client) UpdateCategory(ctx context.Context, planID, categoryID string, category SaveCategory) (Category, error) {
	var data savedCategory
	body := struct {
		Category SaveCategory `json:"category"`
	}{Category: category}
	if err := c.do(ctx, http.MethodPatch, "/plans/"+planID+"/categories/"+categoryID, nil, body, &data); err != nil {
		return Category{}, err
	}
	if data.Category == nil {
		return Category{}, errors.New("the API updated the category but returned no record")
	}
	return *data.Category, nil
}

// savedCategoryGroup is the data of the group create and update responses.
type savedCategoryGroup struct {
	Group *CategoryGroup `json:"category_group"`
}

// CreateCategoryGroup creates a category group and returns it as stored,
// without categories.
func (c *Client) CreateCategoryGroup(ctx context.Context, planID, name string) (CategoryGroup, error) {
	var data savedCategoryGroup
	if err := c.do(ctx, http.MethodPost, "/plans/"+planID+"/category_groups", nil, categoryGroupBody(name), &data); err != nil {
		return CategoryGroup{}, err
	}
	if data.Group == nil {
		return CategoryGroup{}, errors.New("the API created the category group but returned no record")
	}
	return *data.Group, nil
}

// UpdateCategoryGroup renames a category group, the only field the API
// writes, and returns it as stored.
func (c *Client) UpdateCategoryGroup(ctx context.Context, planID, groupID, name string) (CategoryGroup, error) {
	var data savedCategoryGroup
	if err := c.do(ctx, http.MethodPatch, "/plans/"+planID+"/category_groups/"+groupID, nil, categoryGroupBody(name), &data); err != nil {
		return CategoryGroup{}, err
	}
	if data.Group == nil {
		return CategoryGroup{}, errors.New("the API updated the category group but returned no record")
	}
	return *data.Group, nil
}

func categoryGroupBody(name string) any {
	return struct {
		Group struct {
			Name string `json:"name"`
		} `json:"category_group"`
	}{Group: struct {
		Name string `json:"name"`
	}{Name: name}}
}
