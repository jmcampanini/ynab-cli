package cmd

import (
	"time"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// accountRecord is the --jsonl and --csv shape of an account.
type accountRecord struct {
	ID                  string      `json:"id"`
	Name                string      `json:"name"`
	Type                string      `json:"type"`
	OnPlan              bool        `json:"on_plan"`
	Closed              bool        `json:"closed"`
	Note                string      `json:"note,omitempty"`
	Balance             ynab.Amount `json:"balance"`
	ClearedBalance      ynab.Amount `json:"cleared_balance"`
	UnclearedBalance    ynab.Amount `json:"uncleared_balance"`
	TransferPayeeID     string      `json:"transfer_payee_id,omitempty"`
	DirectImportLinked  bool        `json:"direct_import_linked"`
	DirectImportInError bool        `json:"direct_import_in_error"`
	LastReconciledAt    *time.Time  `json:"last_reconciled_at,omitempty"`
}

func newAccountRecord(account ynab.Account) accountRecord {
	record := accountRecord{
		Balance:             account.Balance,
		ClearedBalance:      account.ClearedBalance,
		Closed:              account.Closed,
		DirectImportInError: account.DirectImportInError,
		DirectImportLinked:  account.DirectImportLinked,
		ID:                  account.ID,
		LastReconciledAt:    account.LastReconciledAt,
		Name:                account.Name,
		OnPlan:              account.OnPlan,
		Type:                account.Type,
		UnclearedBalance:    account.UnclearedBalance,
	}
	if account.Note != nil {
		record.Note = *account.Note
	}
	if account.TransferPayeeID != nil {
		record.TransferPayeeID = *account.TransferPayeeID
	}
	return record
}

// kind names the account's role in the plan without saying budget.
func (r accountRecord) kind() string {
	if r.OnPlan {
		return "plan"
	}
	return "tracking"
}

// directImport summarizes the bank link state for human output.
func (r accountRecord) directImport() string {
	switch {
	case r.DirectImportInError:
		return "error"
	case r.DirectImportLinked:
		return "linked"
	}
	return ""
}

func newAccounts(a *app) *cobra.Command {
	command := &cobra.Command{
		Use: "accounts", Short: "List and read the plan's accounts",
		Long: `Read the accounts of the configured plan. A bare 'ynab accounts' prints
this help and exits 0. The API cannot rename, close, delete, or reconcile
an account, so no such verbs exist here.

` + planHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(newAccountsList(a), newAccountsGet(a))
	return command
}
