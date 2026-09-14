package ynab

import "context"

// ScheduledTransaction is a recurring or future transaction with the
// account, payee, and category names filled in. The API cannot create
// split scheduled transactions, but existing ones carry their lines in
// Subtransactions with a nil CategoryID and CategoryName "Split".
type ScheduledTransaction struct {
	AccountID         string                    `json:"account_id"`
	AccountName       string                    `json:"account_name"`
	Amount            Amount                    `json:"amount"`
	CategoryID        *string                   `json:"category_id"`
	CategoryName      *string                   `json:"category_name"`
	DateFirst         string                    `json:"date_first"`
	DateNext          string                    `json:"date_next"`
	Deleted           bool                      `json:"deleted"`
	FlagColor         *string                   `json:"flag_color"`
	FlagName          *string                   `json:"flag_name"`
	Frequency         string                    `json:"frequency"`
	ID                string                    `json:"id"`
	Memo              *string                   `json:"memo"`
	PayeeID           *string                   `json:"payee_id"`
	PayeeName         *string                   `json:"payee_name"`
	Subtransactions   []ScheduledSubtransaction `json:"subtransactions"`
	TransferAccountID *string                   `json:"transfer_account_id"`
}

// ScheduledSubtransaction is one line of a split scheduled transaction.
type ScheduledSubtransaction struct {
	Amount                 Amount  `json:"amount"`
	CategoryID             *string `json:"category_id"`
	CategoryName           *string `json:"category_name"`
	Deleted                bool    `json:"deleted"`
	ID                     string  `json:"id"`
	Memo                   *string `json:"memo"`
	PayeeID                *string `json:"payee_id"`
	PayeeName              *string `json:"payee_name"`
	ScheduledTransactionID string  `json:"scheduled_transaction_id"`
	TransferAccountID      *string `json:"transfer_account_id"`
}

// ScheduledTransactions returns every scheduled transaction in the plan
// in the API's order.
func (c *Client) ScheduledTransactions(ctx context.Context, planID string) ([]ScheduledTransaction, error) {
	var data struct {
		ScheduledTransactions []ScheduledTransaction `json:"scheduled_transactions"`
	}
	if err := c.get(ctx, "/plans/"+planID+"/scheduled_transactions", &data); err != nil {
		return nil, err
	}
	return data.ScheduledTransactions, nil
}

// ScheduledTransaction returns one scheduled transaction by ID.
func (c *Client) ScheduledTransaction(ctx context.Context, planID, scheduledID string) (ScheduledTransaction, error) {
	var data struct {
		ScheduledTransaction ScheduledTransaction `json:"scheduled_transaction"`
	}
	if err := c.get(ctx, "/plans/"+planID+"/scheduled_transactions/"+scheduledID, &data); err != nil {
		return ScheduledTransaction{}, err
	}
	return data.ScheduledTransaction, nil
}
