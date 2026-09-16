package cmd

import (
	"strconv"
	"time"

	"github.com/jmcampanini/ynab-cli/internal/names"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newAccountsGet(a *app) *cobra.Command {
	var output outputFlags
	command := &cobra.Command{
		Use: "get ACCOUNT", Short: "Show every field of one account",
		Long: `Show one account of the configured plan. ACCOUNT is the account ID or
its exact name, matched case-insensitively; closed accounts match too. No
prefix or fuzzy matching: a miss fails and lists the closest names. Two
requests: the plans endpoint, then the accounts endpoint. --jsonl prints
one object.

` + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab accounts get "Chase Checking"
  ynab accounts get 7a3e... --jsonl`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			loaded, client, err := a.connect(cmd)
			if err != nil {
				return err
			}
			plan, err := selectedPlan(cmd.Context(), client, loaded.Config)
			if err != nil {
				return err
			}
			accounts, err := client.Accounts(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}
			account, err := findAccount(args[0], accounts)
			if err != nil {
				return err
			}

			record := newAccountRecord(account)
			out := cmd.OutOrStdout()
			if output.jsonl {
				return writeJSONL(out, []accountRecord{record})
			}
			return writeFields(out, accountFields(record, plan.CurrencyFormat))
		}),
	}
	output.bind(command, false)
	command.ValidArgsFunction = a.completeAccounts()
	return command
}

// accountFields lists the record for human output.
func accountFields(record accountRecord, currency *ynab.CurrencyFormat) [][2]string {
	lastReconciled := ""
	if record.LastReconciledAt != nil {
		lastReconciled = record.LastReconciledAt.UTC().Format(time.DateTime)
	}
	return [][2]string{
		{"id", record.ID},
		{"name", record.Name},
		{"type", record.Type},
		{"kind", record.kind()},
		{"closed", strconv.FormatBool(record.Closed)},
		{"note", record.Note},
		{"balance", record.Balance.Format(currency)},
		{"cleared", record.ClearedBalance.Format(currency)},
		{"uncleared", record.UnclearedBalance.Format(currency)},
		{"transfer payee id", record.TransferPayeeID},
		{"direct import", record.directImport()},
		{"last reconciled", lastReconciled},
	}
}

// findAccount resolves query against account IDs, then names ignoring
// case, closed accounts included.
func findAccount(query string, accounts []ynab.Account) (ynab.Account, error) {
	candidates := make([]names.Candidate, len(accounts))
	for i, account := range accounts {
		candidates[i] = names.Candidate{ID: account.ID, Names: []string{account.Name}}
	}

	index, err := names.Resolve("account", query, candidates)
	if err != nil {
		return ynab.Account{}, err
	}
	return accounts[index], nil
}
