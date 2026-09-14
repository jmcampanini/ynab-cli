package cmd

import (
	"time"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
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
		FromID:  text(movement.FromCategoryID),
		GroupID: text(movement.GroupID),
		ID:      movement.ID,
		MovedAt: movement.MovedAt,
		Note:    text(movement.Note),
		To:      categories.name(movement.ToCategoryID),
		ToID:    text(movement.ToCategoryID),
	}
	if movement.Month != nil {
		record.Month = shortMonth(*movement.Month)
	}
	return record
}

func newMoneyMovements(a *app) *cobra.Command {
	command := &cobra.Command{
		Use: "money-movements", Short: "List the plan's recorded money movements",
		Long: `Read the money movements of the configured plan: each recorded move of
assigned money between two categories, or between a category and ready
to assign. A bare 'ynab money-movements' prints this help and exits 0.
The API records movements but cannot create them; a move is two assigned
amount writes, which arrive in a later release.

` + planHelp + "\n\n" + monthHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(newMoneyMovementsList(a))
	return command
}
