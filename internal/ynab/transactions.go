package ynab

import (
	"context"
	"net/url"
)

// Transaction is one register entry as the transaction endpoints return
// it: the API's transaction detail with the account, payee, and category
// names filled in. A split carries its lines in Subtransactions and has a
// nil CategoryID with CategoryName "Split". Pointer fields are nil when the
// API sends null. FlagColor may also be the empty string for no flag.
type Transaction struct {
	AccountID               string           `json:"account_id"`
	AccountName             string           `json:"account_name"`
	Amount                  Amount           `json:"amount"`
	Approved                bool             `json:"approved"`
	CategoryID              *string          `json:"category_id"`
	CategoryName            *string          `json:"category_name"`
	Cleared                 string           `json:"cleared"`
	Date                    string           `json:"date"`
	DebtTransactionType     *string          `json:"debt_transaction_type"`
	Deleted                 bool             `json:"deleted"`
	FlagColor               *string          `json:"flag_color"`
	FlagName                *string          `json:"flag_name"`
	ID                      string           `json:"id"`
	ImportID                *string          `json:"import_id"`
	ImportPayeeName         *string          `json:"import_payee_name"`
	ImportPayeeNameOriginal *string          `json:"import_payee_name_original"`
	MatchedTransactionID    *string          `json:"matched_transaction_id"`
	Memo                    *string          `json:"memo"`
	PayeeID                 *string          `json:"payee_id"`
	PayeeName               *string          `json:"payee_name"`
	Subtransactions         []Subtransaction `json:"subtransactions"`
	TransferAccountID       *string          `json:"transfer_account_id"`
	TransferTransactionID   *string          `json:"transfer_transaction_id"`
}

// Subtransaction is one line of a split transaction. Lines have no date,
// account, cleared state, or approval of their own; the parent carries
// those.
type Subtransaction struct {
	Amount                Amount  `json:"amount"`
	CategoryID            *string `json:"category_id"`
	CategoryName          *string `json:"category_name"`
	Deleted               bool    `json:"deleted"`
	ID                    string  `json:"id"`
	Memo                  *string `json:"memo"`
	PayeeID               *string `json:"payee_id"`
	PayeeName             *string `json:"payee_name"`
	TransactionID         string  `json:"transaction_id"`
	TransferAccountID     *string `json:"transfer_account_id"`
	TransferTransactionID *string `json:"transfer_transaction_id"`
}

// Transaction types the listing endpoints filter by.
const (
	TypeUnapproved    = "unapproved"
	TypeUncategorized = "uncategorized"
)

// TransactionFilter is the query the transaction listing endpoints accept.
// Dates are YYYY-MM-DD; Type is TypeUnapproved or TypeUncategorized. Empty
// fields are not sent. Without SinceDate the API returns one year back.
type TransactionFilter struct {
	SinceDate string
	Type      string
	UntilDate string
}

func (f TransactionFilter) values() url.Values {
	query := url.Values{}
	if f.SinceDate != "" {
		query.Set("since_date", f.SinceDate)
	}
	if f.UntilDate != "" {
		query.Set("until_date", f.UntilDate)
	}
	if f.Type != "" {
		query.Set("type", f.Type)
	}
	return query
}

// Transactions returns the plan's transactions matching the filter in the
// API's order, oldest first, excluding pending bank transactions.
func (c *Client) Transactions(ctx context.Context, planID string, filter TransactionFilter) ([]Transaction, error) {
	return c.listTransactions(ctx, "/plans/"+planID+"/transactions", filter)
}

// AccountTransactions returns one account's transactions matching the
// filter.
func (c *Client) AccountTransactions(ctx context.Context, planID, accountID string, filter TransactionFilter) ([]Transaction, error) {
	return c.listTransactions(ctx, "/plans/"+planID+"/accounts/"+accountID+"/transactions", filter)
}

// MonthTransactions returns one plan month's transactions matching the
// filter. month must be the API's YYYY-MM-01 form.
func (c *Client) MonthTransactions(ctx context.Context, planID, month string, filter TransactionFilter) ([]Transaction, error) {
	return c.listTransactions(ctx, "/plans/"+planID+"/months/"+month+"/transactions", filter)
}

func (c *Client) listTransactions(ctx context.Context, path string, filter TransactionFilter) ([]Transaction, error) {
	var data struct {
		Transactions []Transaction `json:"transactions"`
	}
	if err := c.getQuery(ctx, path, filter.values(), &data); err != nil {
		return nil, err
	}
	return data.Transactions, nil
}

// Transaction returns one transaction by ID with its lines.
func (c *Client) Transaction(ctx context.Context, planID, transactionID string) (Transaction, error) {
	var data struct {
		Transaction Transaction `json:"transaction"`
	}
	if err := c.get(ctx, "/plans/"+planID+"/transactions/"+transactionID, &data); err != nil {
		return Transaction{}, err
	}
	return data.Transaction, nil
}
