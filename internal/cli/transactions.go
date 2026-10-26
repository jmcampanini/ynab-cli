package cli

import (
	"fmt"
	"io"

	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/selection"
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

func newTransactionRecord(tx ynab.Transaction, accounts selection.AccountNames) transactionRecord {
	record := transactionRecord{
		Account:                 tx.AccountName,
		AccountID:               tx.AccountID,
		Amount:                  tx.Amount,
		Approved:                tx.Approved,
		Category:                render.Text(tx.CategoryName),
		CategoryID:              render.Text(tx.CategoryID),
		Cleared:                 tx.Cleared,
		Date:                    tx.Date,
		DebtTransactionType:     render.Text(tx.DebtTransactionType),
		FlagColor:               render.Text(tx.FlagColor),
		FlagName:                render.Text(tx.FlagName),
		ID:                      tx.ID,
		ImportID:                render.Text(tx.ImportID),
		ImportPayeeName:         render.Text(tx.ImportPayeeName),
		ImportPayeeNameOriginal: render.Text(tx.ImportPayeeNameOriginal),
		MatchedTransactionID:    render.Text(tx.MatchedTransactionID),
		Memo:                    render.Text(tx.Memo),
		Payee:                   render.Text(tx.PayeeName),
		PayeeID:                 render.Text(tx.PayeeID),
		TransferAccount:         accounts.Name(tx.TransferAccountID),
		TransferAccountID:       render.Text(tx.TransferAccountID),
		TransferTransactionID:   render.Text(tx.TransferTransactionID),
	}
	for _, line := range tx.Subtransactions {
		record.Subtransactions = append(record.Subtransactions, subtransactionRecord{
			Amount:                line.Amount,
			Category:              render.Text(line.CategoryName),
			CategoryID:            render.Text(line.CategoryID),
			ID:                    line.ID,
			Memo:                  render.Text(line.Memo),
			Payee:                 render.Text(line.PayeeName),
			PayeeID:               render.Text(line.PayeeID),
			TransferAccount:       accounts.Name(line.TransferAccountID),
			TransferAccountID:     render.Text(line.TransferAccountID),
			TransferTransactionID: render.Text(line.TransferTransactionID),
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

var transactionColumns = []render.Column{
	{Name: "DATE"}, {Name: "ACCOUNT"}, {Name: "PAYEE"}, {Name: "CATEGORY"}, {Name: "MEMO"},
	{Name: "AMOUNT", Right: true}, {Name: "CLEARED"}, {Name: "APPROVED"}, {Name: "FLAG"},
}

// transactionRows renders each transaction as a row, with a split's
// lines beneath it showing only the line's payee, category, memo, and
// amount, the category indented.
func transactionRows(records []transactionRecord, currency *ynab.CurrencyFormat) [][]render.Cell {
	var rows [][]render.Cell
	for _, record := range records {
		rows = append(rows, []render.Cell{
			render.Plain(record.Date), render.Plain(record.Account), render.Plain(record.Payee), render.Plain(record.Category), render.Plain(record.Memo),
			render.Amount(record.Amount, currency), render.Plain(record.Cleared), render.Plain(yesNo(record.Approved)), render.Plain(record.FlagColor),
		})
		for _, line := range record.Subtransactions {
			rows = append(rows, []render.Cell{
				render.Plain(""), render.Plain(""), render.Plain(line.Payee), render.Plain("  " + line.Category), render.Plain(line.Memo),
				render.Amount(line.Amount, currency), render.Plain(""), render.Plain(""), render.Plain(""),
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

func newTransactions(a *app) *cobra.Command {
	command := &cobra.Command{
		Use: "transactions", Short: "Read, review, and write the plan's transactions",
		Long: `Read and write the register of the configured plan: the transaction
list with its filters, one transaction by ID, the review of what needs
approval or a category, and the writes that carry the monthly work. A
bare 'ynab transactions' prints this help and exits 0. The API cannot
edit the lines of an existing split, reconcile an account, or read
pending bank transactions. Converting an existing transaction to a split
is available through 'ynab api put', but not through this group.

` + writeHelp + "\n\n" + planHelp + "\n\n" + dateHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(newTransactionsList(a), newTransactionsGet(a), newTransactionsReview(a), newTransactionsCreate(a), newTransactionsUpdate(a), newTransactionsDelete(a), newTransactionsApprove(a), newTransactionsCategorize(a), newTransactionsClear(a), newTransactionsUnclear(a), newTransactionsFlag(a), newTransactionsImport(a))
	return command
}

// writeTransactionFields renders one transaction as a field listing with,
// for a split, its lines in a table beneath.
func writeTransactionFields(out io.Writer, record transactionRecord, currency *ynab.CurrencyFormat) error {
	err := render.Fields(out, [][2]string{
		{"id", record.ID},
		{"date", record.Date},
		{"account", record.Account},
		{"payee", record.Payee},
		{"category", record.Category},
		{"memo", record.Memo},
		{"amount", record.Amount.Format(currency)},
		{"cleared", record.Cleared},
		{"approved", yesNo(record.Approved)},
		{"flag", render.FlagWords(record.FlagColor, record.FlagName)},
		{"transfer account", record.TransferAccount},
		{"transfer transaction", record.TransferTransactionID},
		{"matched transaction", record.MatchedTransactionID},
		{"import id", record.ImportID},
		{"import payee", record.ImportPayeeName},
		{"import payee original", record.ImportPayeeNameOriginal},
		{"debt type", record.DebtTransactionType},
	})
	if err != nil || len(record.Subtransactions) == 0 {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	rows := make([][]render.Cell, len(record.Subtransactions))
	for i, line := range record.Subtransactions {
		rows[i] = render.SplitRow(line.Payee, line.Category, line.Memo, line.Amount, currency)
	}
	return render.Table(out, render.SplitColumns, rows)
}

// Help paragraphs only transactions members use.
const dateFlagHelp = `Dates:
  --date accepts YYYY-MM-DD, today, or yesterday. today and yesterday
  follow the local clock. Any other form is a usage error.`

const bulkHelp = `Bulk updates:
  IDs are sent in one bulk update per 100 transactions, a limit this CLI
  chooses; past one batch, progress goes to stderr. If a later batch
  fails, the records already applied are printed before the error with
  the IDs not changed, and the exit status is 1. A dry run, or a check
  that needs the current records, reads them with one plan listing
  request plus one request per ID older than the API's one-year default
  window; a dry run prints them with the change applied.`

// Flag values for --cleared and --flag.
var (
	clearedStates = []string{"uncleared", "cleared", "reconciled"}
	flagColors    = []string{"red", "orange", "yellow", "green", "blue", "purple", "none"}
)
