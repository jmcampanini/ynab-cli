package cli

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/selection"
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
			plan, err := selection.Plan(cmd.Context(), client, loaded.Config)
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

			records = limitRecords(&output, records)
			out := cmd.OutOrStdout()
			if output.machine() {
				return writeRecords(out, output, records)
			}
			if err := render.Table(out, accountColumns, accountRows(records, plan.CurrencyFormat, a.palette(cmd))); err != nil {
				return err
			}
			summary := render.CountNoun(len(records), "account", "accounts")
			if hidden > 0 {
				summary += fmt.Sprintf(", %d closed hidden", hidden)
			}
			_, err = fmt.Fprintln(out, summary)
			return err
		}),
	}
	output.bindRead(command, true)
	command.Flags().BoolVar(&includeClosed, "closed", false, "Include closed accounts")
	return command
}

var accountColumns = []render.Column{
	{Name: "NAME"}, {Name: "TYPE"}, {Name: "KIND"},
	{Name: "BALANCE", Right: true}, {Name: "CLEARED", Right: true}, {Name: "UNCLEARED", Right: true},
	{Name: "IMPORT"},
}

func accountRows(records []accountRecord, currency *ynab.CurrencyFormat, colors render.Palette) [][]render.Cell {
	rows := make([][]render.Cell, len(records))
	for i, record := range records {
		name := record.Name
		if record.Closed {
			name += " (closed)"
		}
		row := []render.Cell{
			render.Plain(name), render.Plain(record.Type), render.Plain(record.kind()),
			render.Available(record.Balance, currency, colors),
			render.Available(record.ClearedBalance, currency, colors),
			render.Available(record.UnclearedBalance, currency, colors),
			render.Plain(record.directImport()),
		}
		if record.Closed {
			for j := range row {
				row[j].Paint = colors.Faint
			}
		}
		rows[i] = row
	}
	return rows
}
