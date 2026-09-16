package ynab

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
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

// Nullable is a request field with three states: unset, which leaves the
// field out of the body so an update keeps the current value; null, which
// clears it; and a value. Declare it with the omitzero JSON option.
type Nullable[T any] struct {
	// Set reports whether the field is sent at all.
	Set bool
	// Value is the value to send, or nil for JSON null.
	Value *T
}

// NullValue returns a Nullable that sends null.
func NullValue[T any]() Nullable[T] { return Nullable[T]{Set: true} }

// SomeValue returns a Nullable that sends value.
func SomeValue[T any](value T) Nullable[T] { return Nullable[T]{Set: true, Value: &value} }

// IsZero reports whether the field is unset, which omitzero uses to leave
// it out of the body.
func (n Nullable[T]) IsZero() bool { return !n.Set }

// MarshalJSON writes null or the value.
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if n.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*n.Value)
}

// SaveTransaction is the body of a create or of one entry in a bulk
// update. A nil pointer or empty string is left out of the body, so an
// update changes only the fields it names. Milliunits carries the amount
// as the API's integer, since the CLI's decimal Amount encoding is for
// output. ID names the transaction in a bulk update and is empty on
// create. PayeeName resolves an existing payee by name or creates one,
// and is ignored when PayeeID is set.
type SaveTransaction struct {
	AccountID       string               `json:"account_id,omitempty"`
	Approved        *bool                `json:"approved,omitempty"`
	CategoryID      *string              `json:"category_id,omitempty"`
	Cleared         string               `json:"cleared,omitempty"`
	Date            string               `json:"date,omitempty"`
	FlagColor       Nullable[string]     `json:"flag_color,omitzero"`
	ID              string               `json:"id,omitempty"`
	ImportID        string               `json:"import_id,omitempty"`
	Memo            *string              `json:"memo,omitempty"`
	Milliunits      *int64               `json:"amount,omitempty"`
	PayeeID         *string              `json:"payee_id,omitempty"`
	PayeeName       *string              `json:"payee_name,omitempty"`
	Subtransactions []SaveSubtransaction `json:"subtransactions,omitempty"`
}

// SaveSubtransaction is one line of a split on create. The API cannot
// change the lines of an existing split.
type SaveSubtransaction struct {
	CategoryID *string `json:"category_id,omitempty"`
	Memo       *string `json:"memo,omitempty"`
	Milliunits int64   `json:"amount"`
	PayeeID    *string `json:"payee_id,omitempty"`
	PayeeName  *string `json:"payee_name,omitempty"`
}

// SetAmount records amount as the milliunits to send.
func (s *SaveTransaction) SetAmount(amount Amount) {
	milliunits := int64(amount)
	s.Milliunits = &milliunits
}

// savedTransactions is the data of the create and bulk update responses.
// The CLI uses the single-transaction create, which answers a duplicate
// import ID with a 409 rather than the duplicate_import_ids list.
type savedTransactions struct {
	Transaction  *Transaction  `json:"transaction"`
	Transactions []Transaction `json:"transactions"`
}

// CreateTransaction creates one transaction and returns it as stored,
// with names filled in. A duplicate import ID is a 409 conflict.
func (c *Client) CreateTransaction(ctx context.Context, planID string, tx SaveTransaction) (Transaction, error) {
	var data savedTransactions
	body := struct {
		Transaction SaveTransaction `json:"transaction"`
	}{Transaction: tx}
	if err := c.do(ctx, http.MethodPost, "/plans/"+planID+"/transactions", nil, body, &data); err != nil {
		return Transaction{}, err
	}
	if data.Transaction == nil {
		return Transaction{}, errors.New("the API created the transaction but returned no record")
	}
	return *data.Transaction, nil
}

// UpdateTransactions applies each change to the transaction its ID names
// in one request and returns the stored records in the API's order. The
// API states no limit on the count; the CLI batches on its own.
func (c *Client) UpdateTransactions(ctx context.Context, planID string, changes []SaveTransaction) ([]Transaction, error) {
	var data savedTransactions
	body := struct {
		Transactions []SaveTransaction `json:"transactions"`
	}{Transactions: changes}
	if err := c.do(ctx, http.MethodPatch, "/plans/"+planID+"/transactions", nil, body, &data); err != nil {
		return nil, err
	}
	return data.Transactions, nil
}

// DeleteTransaction deletes one transaction and returns it as it was.
func (c *Client) DeleteTransaction(ctx context.Context, planID, transactionID string) (Transaction, error) {
	var data struct {
		Transaction *Transaction `json:"transaction"`
	}
	if err := c.do(ctx, http.MethodDelete, "/plans/"+planID+"/transactions/"+transactionID, nil, nil, &data); err != nil {
		return Transaction{}, err
	}
	if data.Transaction == nil {
		return Transaction{}, errors.New("the API deleted the transaction but returned no record")
	}
	return *data.Transaction, nil
}

// ImportTransactions asks the API to import from every linked account and
// returns the IDs of the transactions it imported, which may be none.
func (c *Client) ImportTransactions(ctx context.Context, planID string) ([]string, error) {
	var data struct {
		TransactionIDs *[]string `json:"transaction_ids"`
	}
	if err := c.do(ctx, http.MethodPost, "/plans/"+planID+"/transactions/import", nil, nil, &data); err != nil {
		return nil, err
	}
	if data.TransactionIDs == nil {
		return nil, errors.New("the API answered the import but returned no transaction_ids")
	}
	return *data.TransactionIDs, nil
}
