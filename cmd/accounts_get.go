package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jmcampanini/ynab-cli/internal/names"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// closestNamesShown bounds the suggestions in a failed lookup.
const closestNamesShown = 5

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
			lastReconciled := ""
			if record.LastReconciledAt != nil {
				lastReconciled = record.LastReconciledAt.UTC().Format(time.DateTime)
			}
			return writeFields(out, [][2]string{
				{"id", record.ID},
				{"name", record.Name},
				{"type", record.Type},
				{"kind", record.kind()},
				{"closed", strconv.FormatBool(record.Closed)},
				{"note", record.Note},
				{"balance", record.Balance.Format(plan.CurrencyFormat)},
				{"cleared", record.ClearedBalance.Format(plan.CurrencyFormat)},
				{"uncleared", record.UnclearedBalance.Format(plan.CurrencyFormat)},
				{"transfer payee id", record.TransferPayeeID},
				{"direct import", record.directImport()},
				{"last reconciled", lastReconciled},
			})
		}),
	}
	output.bind(command, false)
	return command
}

// findAccount matches query against account IDs, then names ignoring
// case. A miss lists the closest names; several name matches list them.
func findAccount(query string, accounts []ynab.Account) (ynab.Account, error) {
	accountNames := make([]string, len(accounts))
	for i, account := range accounts {
		if account.ID == query {
			return account, nil
		}
		accountNames[i] = account.Name
	}

	matches := names.Match(query, accountNames)
	switch len(matches) {
	case 1:
		return accounts[matches[0]], nil
	case 0:
		closest := names.Closest(query, accountNames, closestNamesShown)
		if len(closest) == 0 {
			return ynab.Account{}, fmt.Errorf("account %q not found; the plan has no accounts", query)
		}
		return ynab.Account{}, fmt.Errorf("account %q not found; closest names: %s", query, quoteAll(closest))
	default:
		matched := make([]string, len(matches))
		for i, index := range matches {
			matched[i] = accounts[index].ID
		}
		return ynab.Account{}, fmt.Errorf("account %q matches %d accounts; use an ID: %s", query, len(matches), strings.Join(matched, ", "))
	}
}

func quoteAll(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = strconv.Quote(value)
	}
	return strings.Join(quoted, ", ")
}
