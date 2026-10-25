package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

func (s *writeSession) repairSplitDeletion(ctx context.Context, tx ynab.Transaction, diagnostics io.Writer) error {
	if tx.Deleted || tx.TransferAccountID != nil || !s.accountByID(tx.AccountID).OnPlan {
		return nil
	}
	seen := map[string]bool{}
	for _, line := range tx.Subtransactions {
		if line.Deleted || line.CategoryID == nil || *line.CategoryID == "" || line.TransferAccountID != nil || seen[*line.CategoryID] {
			continue
		}
		category := ynab.Category{ID: *line.CategoryID, Name: render.Text(line.CategoryName)}
		if err := s.recalculateSplitCategory(ctx, tx, category); err != nil {
			return err
		}
		seen[category.ID] = true
		if _, err := fmt.Fprintf(diagnostics, "warning [split-delete-recalculated]: category %q (%s), transaction date %s: created and deleted a zero-amount entry after split deletion\n",
			category.Name, category.ID, tx.Date); err != nil {
			return fmt.Errorf("report category recalculation: %w", err)
		}
	}
	return nil
}

func (s *writeSession) recalculateSplitCategory(ctx context.Context, tx ynab.Transaction, category ynab.Category) error {
	memo := "ynab-cli: recalculate category after deleting split " + tx.ID
	approved := false
	entry := ynab.SaveTransaction{
		AccountID:  tx.AccountID,
		Approved:   &approved,
		CategoryID: &category.ID,
		Cleared:    "uncleared",
		Date:       tx.Date,
		Memo:       &memo,
	}
	entry.SetAmount(0)
	temporary, err := s.client.CreateTransaction(ctx, s.plan.ID, entry)
	if err != nil {
		return fmt.Errorf("create temporary entry for category %q (%s); inspect the register for memo %q before retrying: %w", category.Name, category.ID, memo, err)
	}
	if temporary.ID == "" {
		return fmt.Errorf("temporary entry for category %q (%s) returned no ID; inspect the register for memo %q", category.Name, category.ID, memo)
	}

	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	if _, err := s.client.DeleteTransaction(cleanupCtx, s.plan.ID, temporary.ID); err != nil {
		return fmt.Errorf("remove temporary zero-amount transaction %s for category %q (%s); it may remain in the register: %w", temporary.ID, category.Name, category.ID, err)
	}
	return nil
}
