package cmd

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newAccountsList(a *app) *cobra.Command {
	var output outputFlags
	var includeClosed bool
	command := &cobra.Command{
		Use: "list", Short: "List the plan's accounts with their balances",
		Long: `List the configured plan's accounts: name, type, whether the account is
on the plan or a tracking account, the balance, cleared, and uncleared
amounts, and the direct import state (linked, error, or blank). Closed
accounts are hidden unless --closed is given, which shows them with a
"(closed)" suffix. Negative balances are red and closed accounts faint when
color is on. Two requests: the plans endpoint, then the accounts endpoint.

` + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab accounts list
  ynab accounts list --closed
  ynab accounts list --jsonl | jq .balance
  ynab accounts list --csv > accounts.csv`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
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

			var records []accountRecord
			hidden := 0
			for _, account := range accounts {
				if account.Closed && !includeClosed {
					hidden++
					continue
				}
				records = append(records, newAccountRecord(account))
			}

			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return writeJSONL(out, records)
			case output.csv:
				return writeCSV(out, records)
			}
			if err := writeTable(out, accountColumns, accountRows(records, plan.CurrencyFormat, a.palette(cmd))); err != nil {
				return err
			}
			summary := countNoun(len(records), "account", "accounts")
			if hidden > 0 {
				summary += fmt.Sprintf(", %d closed hidden", hidden)
			}
			_, err = fmt.Fprintln(out, summary)
			return err
		}),
	}
	output.bind(command, true)
	command.Flags().BoolVar(&includeClosed, "closed", false, "Include closed accounts")
	return command
}

var accountColumns = []column{
	{name: "NAME"}, {name: "TYPE"}, {name: "KIND"},
	{name: "BALANCE", right: true}, {name: "CLEARED", right: true}, {name: "UNCLEARED", right: true},
	{name: "IMPORT"},
}

func accountRows(records []accountRecord, currency *ynab.CurrencyFormat, colors palette) [][]cell {
	rows := make([][]cell, len(records))
	for i, record := range records {
		name := record.Name
		if record.Closed {
			name += " (closed)"
		}
		row := []cell{
			plain(name), plain(record.Type), plain(record.kind()),
			amountCell(record.Balance, currency, colors),
			amountCell(record.ClearedBalance, currency, colors),
			amountCell(record.UnclearedBalance, currency, colors),
			plain(record.directImport()),
		}
		if record.Closed {
			for j := range row {
				row[j].paint = colors.faint
			}
		}
		rows[i] = row
	}
	return rows
}

func amountCell(amount ynab.Amount, currency *ynab.CurrencyFormat, colors palette) cell {
	c := plain(amount.Format(currency))
	if amount < 0 {
		c.paint = colors.red
	}
	return c
}
