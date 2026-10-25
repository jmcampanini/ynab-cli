package cli

import (
	"time"

	"github.com/jmcampanini/ynab-cli/internal/dates"
	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// readyToAssign names the source or target of a move that the API records
// with no category.
const readyToAssign = "Ready to Assign"

// moneyMovementRecord is the --jsonl and --csv shape of a money movement.
type moneyMovementRecord struct {
	ID      string      `json:"id"`
	Month   string      `json:"month,omitempty"`
	MovedAt *time.Time  `json:"moved_at,omitempty"`
	From    string      `json:"from"`
	FromID  string      `json:"from_id,omitempty"`
	To      string      `json:"to"`
	ToID    string      `json:"to_id,omitempty"`
	Amount  ynab.Amount `json:"amount"`
	Note    string      `json:"note,omitempty"`
	GroupID string      `json:"group_id,omitempty"`
}

// categoryNames maps category IDs to bare names, hidden ones included.
type categoryNames map[string]string

func newCategoryNames(groups []ynab.CategoryGroup) categoryNames {
	byID := categoryNames{}
	for _, group := range groups {
		for _, category := range group.Categories {
			byID[category.ID] = category.Name
		}
	}
	return byID
}

// name returns the category's name, the ID itself when unknown, and
// readyToAssign for nil.
func (n categoryNames) name(id *string) string {
	if id == nil {
		return readyToAssign
	}
	if name, ok := n[*id]; ok {
		return name
	}
	return *id
}

func newMoneyMovementRecord(movement ynab.MoneyMovement, categories categoryNames) moneyMovementRecord {
	record := moneyMovementRecord{
		Amount:  movement.Amount,
		From:    categories.name(movement.FromCategoryID),
		FromID:  render.Text(movement.FromCategoryID),
		GroupID: render.Text(movement.GroupID),
		ID:      movement.ID,
		MovedAt: movement.MovedAt,
		Note:    render.Text(movement.Note),
		To:      categories.name(movement.ToCategoryID),
		ToID:    render.Text(movement.ToCategoryID),
	}
	if movement.Month != nil {
		record.Month = dates.ShortMonth(*movement.Month)
	}
	return record
}
