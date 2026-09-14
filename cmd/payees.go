package cmd

import (
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// payeeRecord is the --jsonl and --csv shape of a payee. A transfer payee
// stands for an account and names it in TransferAccount.
type payeeRecord struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	TransferAccount   string `json:"transfer_account,omitempty"`
	TransferAccountID string `json:"transfer_account_id,omitempty"`
}

func newPayeeRecord(payee ynab.Payee, accounts accountNames) payeeRecord {
	return payeeRecord{
		ID:                payee.ID,
		Name:              payee.Name,
		TransferAccount:   accounts.name(payee.TransferAccountID),
		TransferAccountID: text(payee.TransferAccountID),
	}
}

func newPayees(a *app) *cobra.Command {
	command := &cobra.Command{
		Use: "payees", Short: "List and read the plan's payees",
		Long: `Read the payees of the configured plan. A bare 'ynab payees' prints this
help and exits 0. Writes arrive in a later release; the API cannot delete
or merge payees, and payee locations are out of scope.

` + planHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(newPayeesList(a), newPayeesGet(a))
	return command
}
