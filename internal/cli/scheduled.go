package cli

import (
	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/selection"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// scheduledRecord is the --jsonl and --csv shape of a scheduled
// transaction. A split carries its lines in Subtransactions; CSV flattens
// them to one row per line with ParentID set.
type scheduledRecord struct {
	ID                string                          `json:"id"`
	ParentID          string                          `json:"parent_id,omitempty"`
	NextDate          string                          `json:"next_date"`
	FirstDate         string                          `json:"first_date"`
	Frequency         string                          `json:"frequency"`
	Account           string                          `json:"account"`
	AccountID         string                          `json:"account_id"`
	Payee             string                          `json:"payee,omitempty"`
	PayeeID           string                          `json:"payee_id,omitempty"`
	Category          string                          `json:"category,omitempty"`
	CategoryID        string                          `json:"category_id,omitempty"`
	Memo              string                          `json:"memo,omitempty"`
	Amount            ynab.Amount                     `json:"amount"`
	FlagColor         string                          `json:"flag_color,omitempty"`
	FlagName          string                          `json:"flag_name,omitempty"`
	TransferAccount   string                          `json:"transfer_account,omitempty"`
	TransferAccountID string                          `json:"transfer_account_id,omitempty"`
	Subtransactions   []scheduledSubtransactionRecord `json:"subtransactions,omitempty"`
}

// scheduledSubtransactionRecord is one line of a split scheduled
// transaction.
type scheduledSubtransactionRecord struct {
	ID                string      `json:"id"`
	Payee             string      `json:"payee,omitempty"`
	PayeeID           string      `json:"payee_id,omitempty"`
	Category          string      `json:"category,omitempty"`
	CategoryID        string      `json:"category_id,omitempty"`
	Memo              string      `json:"memo,omitempty"`
	Amount            ynab.Amount `json:"amount"`
	TransferAccount   string      `json:"transfer_account,omitempty"`
	TransferAccountID string      `json:"transfer_account_id,omitempty"`
}

func newScheduledRecord(scheduled ynab.ScheduledTransaction, accounts selection.AccountNames) scheduledRecord {
	record := scheduledRecord{
		Account:           scheduled.AccountName,
		AccountID:         scheduled.AccountID,
		Amount:            scheduled.Amount,
		Category:          render.Text(scheduled.CategoryName),
		CategoryID:        render.Text(scheduled.CategoryID),
		FirstDate:         scheduled.DateFirst,
		FlagColor:         render.Text(scheduled.FlagColor),
		FlagName:          render.Text(scheduled.FlagName),
		Frequency:         scheduled.Frequency,
		ID:                scheduled.ID,
		Memo:              render.Text(scheduled.Memo),
		NextDate:          scheduled.DateNext,
		Payee:             render.Text(scheduled.PayeeName),
		PayeeID:           render.Text(scheduled.PayeeID),
		TransferAccount:   accounts.Name(scheduled.TransferAccountID),
		TransferAccountID: render.Text(scheduled.TransferAccountID),
	}
	for _, line := range scheduled.Subtransactions {
		record.Subtransactions = append(record.Subtransactions, scheduledSubtransactionRecord{
			Amount:            line.Amount,
			Category:          render.Text(line.CategoryName),
			CategoryID:        render.Text(line.CategoryID),
			ID:                line.ID,
			Memo:              render.Text(line.Memo),
			Payee:             render.Text(line.PayeeName),
			PayeeID:           render.Text(line.PayeeID),
			TransferAccount:   accounts.Name(line.TransferAccountID),
			TransferAccountID: render.Text(line.TransferAccountID),
		})
	}
	return record
}

func newScheduled(a *app) *cobra.Command {
	command := &cobra.Command{
		Use: "scheduled", Short: "List and read the plan's scheduled transactions",
		Long: `Read the scheduled transactions of the configured plan: the recurring
and future transactions YNAB enters on their next date. A bare
'ynab scheduled' prints this help and exits 0. The CLI does not create,
change, or delete scheduled transactions; it reads them and lets the
transactions they enter be imported. The API cannot create split
scheduled transactions, though it returns the lines of existing ones.

` + planHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(newScheduledList(a), newScheduledGet(a))
	return command
}
