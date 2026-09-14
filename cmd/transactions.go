package cmd

import (
	"fmt"
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/dates"
	"github.com/jmcampanini/ynab-cli/internal/names"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// transactionRecord is the --jsonl and --csv shape of a transaction. A
// split carries its lines in Subtransactions; CSV flattens them to one
// row per line with ParentID set, so ParentID is empty in JSONL.
type transactionRecord struct {
	ID                      string                 `json:"id"`
	ParentID                string                 `json:"parent_id,omitempty"`
	Date                    string                 `json:"date"`
	Account                 string                 `json:"account"`
	AccountID               string                 `json:"account_id"`
	Payee                   string                 `json:"payee,omitempty"`
	PayeeID                 string                 `json:"payee_id,omitempty"`
	Category                string                 `json:"category,omitempty"`
	CategoryID              string                 `json:"category_id,omitempty"`
	Memo                    string                 `json:"memo,omitempty"`
	Amount                  ynab.Amount            `json:"amount"`
	Cleared                 string                 `json:"cleared"`
	Approved                bool                   `json:"approved"`
	FlagColor               string                 `json:"flag_color,omitempty"`
	FlagName                string                 `json:"flag_name,omitempty"`
	TransferAccount         string                 `json:"transfer_account,omitempty"`
	TransferAccountID       string                 `json:"transfer_account_id,omitempty"`
	TransferTransactionID   string                 `json:"transfer_transaction_id,omitempty"`
	MatchedTransactionID    string                 `json:"matched_transaction_id,omitempty"`
	ImportID                string                 `json:"import_id,omitempty"`
	ImportPayeeName         string                 `json:"import_payee_name,omitempty"`
	ImportPayeeNameOriginal string                 `json:"import_payee_name_original,omitempty"`
	DebtTransactionType     string                 `json:"debt_transaction_type,omitempty"`
	Subtransactions         []subtransactionRecord `json:"subtransactions,omitempty"`
}

// subtransactionRecord is one line of a split in the parent's
// subtransactions array.
type subtransactionRecord struct {
	ID                    string      `json:"id"`
	Payee                 string      `json:"payee,omitempty"`
	PayeeID               string      `json:"payee_id,omitempty"`
	Category              string      `json:"category,omitempty"`
	CategoryID            string      `json:"category_id,omitempty"`
	Memo                  string      `json:"memo,omitempty"`
	Amount                ynab.Amount `json:"amount"`
	TransferAccount       string      `json:"transfer_account,omitempty"`
	TransferAccountID     string      `json:"transfer_account_id,omitempty"`
	TransferTransactionID string      `json:"transfer_transaction_id,omitempty"`
}

// accountNames maps account IDs to names so transfer targets, which the
// transaction endpoints identify only by ID, can be named.
type accountNames map[string]string

func newAccountNames(accounts []ynab.Account) accountNames {
	byID := make(accountNames, len(accounts))
	for _, account := range accounts {
		byID[account.ID] = account.Name
	}
	return byID
}

// name returns the account's name, the ID itself when the account is
// unknown, and blank for nil.
func (n accountNames) name(id *string) string {
	if id == nil {
		return ""
	}
	if name, ok := n[*id]; ok {
		return name
	}
	return *id
}

// text dereferences an optional API string, blank when null.
func text(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func newTransactionRecord(tx ynab.Transaction, accounts accountNames) transactionRecord {
	record := transactionRecord{
		Account:                 tx.AccountName,
		AccountID:               tx.AccountID,
		Amount:                  tx.Amount,
		Approved:                tx.Approved,
		Category:                text(tx.CategoryName),
		CategoryID:              text(tx.CategoryID),
		Cleared:                 tx.Cleared,
		Date:                    tx.Date,
		DebtTransactionType:     text(tx.DebtTransactionType),
		FlagColor:               text(tx.FlagColor),
		FlagName:                text(tx.FlagName),
		ID:                      tx.ID,
		ImportID:                text(tx.ImportID),
		ImportPayeeName:         text(tx.ImportPayeeName),
		ImportPayeeNameOriginal: text(tx.ImportPayeeNameOriginal),
		MatchedTransactionID:    text(tx.MatchedTransactionID),
		Memo:                    text(tx.Memo),
		Payee:                   text(tx.PayeeName),
		PayeeID:                 text(tx.PayeeID),
		TransferAccount:         accounts.name(tx.TransferAccountID),
		TransferAccountID:       text(tx.TransferAccountID),
		TransferTransactionID:   text(tx.TransferTransactionID),
	}
	for _, line := range tx.Subtransactions {
		record.Subtransactions = append(record.Subtransactions, subtransactionRecord{
			Amount:                line.Amount,
			Category:              text(line.CategoryName),
			CategoryID:            text(line.CategoryID),
			ID:                    line.ID,
			Memo:                  text(line.Memo),
			Payee:                 text(line.PayeeName),
			PayeeID:               text(line.PayeeID),
			TransferAccount:       accounts.name(line.TransferAccountID),
			TransferAccountID:     text(line.TransferAccountID),
			TransferTransactionID: text(line.TransferTransactionID),
		})
	}
	return record
}

// csvLines flattens splits for CSV: each parent row is followed by one
// row per line carrying the parent's ID in parent_id and the parent's
// date, account, cleared, approved, and flag.
func csvLines(records []transactionRecord) []transactionRecord {
	var rows []transactionRecord
	for _, record := range records {
		parent := record
		parent.Subtransactions = nil
		rows = append(rows, parent)
		for _, line := range record.Subtransactions {
			rows = append(rows, transactionRecord{
				Account: parent.Account, AccountID: parent.AccountID, Approved: parent.Approved,
				Cleared: parent.Cleared, Date: parent.Date, FlagColor: parent.FlagColor, FlagName: parent.FlagName,
				Amount: line.Amount, Category: line.Category, CategoryID: line.CategoryID, ID: line.ID,
				Memo: line.Memo, ParentID: parent.ID, Payee: line.Payee, PayeeID: line.PayeeID,
				TransferAccount: line.TransferAccount, TransferAccountID: line.TransferAccountID,
				TransferTransactionID: line.TransferTransactionID,
			})
		}
	}
	return rows
}

var transactionColumns = []column{
	{name: "DATE"}, {name: "ACCOUNT"}, {name: "PAYEE"}, {name: "CATEGORY"}, {name: "MEMO"},
	{name: "AMOUNT", right: true}, {name: "CLEARED"}, {name: "APPROVED"}, {name: "FLAG"},
}

// transactionRows renders each transaction as a row, with a split's
// lines beneath it showing only the line's payee, category, memo, and
// amount, the category indented.
func transactionRows(records []transactionRecord, currency *ynab.CurrencyFormat) [][]cell {
	var rows [][]cell
	for _, record := range records {
		rows = append(rows, []cell{
			plain(record.Date), plain(record.Account), plain(record.Payee), plain(record.Category), plain(record.Memo),
			plainAmount(record.Amount, currency), plain(record.Cleared), plain(yesNo(record.Approved)), plain(record.FlagColor),
		})
		for _, line := range record.Subtransactions {
			rows = append(rows, []cell{
				plain(""), plain(""), plain(line.Payee), plain("  " + line.Category), plain(line.Memo),
				plainAmount(line.Amount, currency), plain(""), plain(""), plain(""),
			})
		}
	}
	return rows
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

// flagWords joins a flag's color and custom name for human output.
func flagWords(color, name string) string {
	if color == "" {
		return ""
	}
	if name == "" {
		return color
	}
	return color + " (" + name + ")"
}

// lineColumns and lineRows render a split's lines under a single record.
var lineColumns = []column{{name: "PAYEE"}, {name: "CATEGORY"}, {name: "MEMO"}, {name: "AMOUNT", right: true}}

func lineRows(lines []subtransactionRecord, currency *ynab.CurrencyFormat) [][]cell {
	rows := make([][]cell, len(lines))
	for i, line := range lines {
		rows[i] = []cell{plain(line.Payee), plain(line.Category), plain(line.Memo), plainAmount(line.Amount, currency)}
	}
	return rows
}

// findPayee resolves query against payee IDs, then names ignoring case.
func findPayee(query string, payees []ynab.Payee) (ynab.Payee, error) {
	candidates := make([]names.Candidate, len(payees))
	for i, payee := range payees {
		candidates[i] = names.Candidate{ID: payee.ID, Names: []string{payee.Name}}
	}

	index, err := names.Resolve("payee", query, candidates)
	if err != nil {
		return ynab.Payee{}, err
	}
	return payees[index], nil
}

// date parses a date operand against the injected clock. A bad form is a
// usage error.
func (a *app) date(text string) (string, error) {
	date, err := dates.Date(text, a.deps.now())
	if err != nil {
		return "", usageError(err.Error())
	}
	return date, nil
}

// containsFold reports whether text contains substring ignoring case.
func containsFold(text, substring string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substring))
}

func transactionCount(count int) string {
	return countNoun(count, "transaction", "transactions")
}

func newTransactions(a *app) *cobra.Command {
	command := &cobra.Command{
		Use: "transactions", Short: "List, read, and review the plan's transactions",
		Long: `Read the register of the configured plan: the transaction list with its
filters, one transaction by ID, and the review of what needs approval or
a category. A bare 'ynab transactions' prints this help and exits 0.
Writes arrive in a later release; the API cannot edit the lines of an
existing split or read pending bank transactions.

` + planHelp + "\n\n" + dateHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(newTransactionsList(a), newTransactionsGet(a), newTransactionsReview(a))
	return command
}

// transactionSummary is the count line under a transaction table: the
// count and the net total, or the total of the matching lines when a
// category or payee filter selected them.
func transactionSummary(count int, total ynab.Amount, matchingLines bool, currency *ynab.CurrencyFormat) string {
	if matchingLines {
		return fmt.Sprintf("%s, matching lines total %s", transactionCount(count), total.Format(currency))
	}
	return fmt.Sprintf("%s, total %s", transactionCount(count), total.Format(currency))
}
