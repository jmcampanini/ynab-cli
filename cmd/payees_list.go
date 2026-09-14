package cmd

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newPayeesList(a *app) *cobra.Command {
	var output outputFlags
	var unused bool
	command := &cobra.Command{
		Use: "list", Short: "List the plan's payees",
		Long: `List the configured plan's payees in the API's order. A transfer payee
shows the account it stands for. Three requests: the plans endpoint, the
accounts endpoint (which names transfer targets), then the payees
endpoint. --unused keeps payees that no transaction references, on the
transaction itself or a split line, and adds a fourth request for every
transaction since the plan's first month; scheduled transactions are not
consulted.

` + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab payees list
  ynab payees list --unused
  ynab payees list --jsonl | jq -r .name`,
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
			payees, err := client.Payees(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}
			var used map[string]bool
			if unused {
				transactions, err := client.Transactions(cmd.Context(), plan.ID, ynab.TransactionFilter{SinceDate: plan.FirstMonth})
				if err != nil {
					return err
				}
				used = usedPayees(transactions)
			}

			names := newAccountNames(accounts)
			var records []payeeRecord
			for _, payee := range payees {
				if unused && used[payee.ID] {
					continue
				}
				records = append(records, newPayeeRecord(payee, names))
			}

			out := cmd.OutOrStdout()
			switch {
			case output.jsonl:
				return writeJSONL(out, records)
			case output.csv:
				return writeCSV(out, records)
			}
			rows := make([][]cell, len(records))
			for i, record := range records {
				rows[i] = []cell{plain(record.Name), plain(record.TransferAccount)}
			}
			if err := writeTable(out, []column{{name: "NAME"}, {name: "TRANSFER ACCOUNT"}}, rows); err != nil {
				return err
			}
			summary := countNoun(len(records), "payee", "payees")
			if unused {
				summary = countNoun(len(records), "unused payee", "unused payees")
			}
			_, err = fmt.Fprintln(out, summary)
			return err
		}),
	}
	output.bind(command, true)
	command.Flags().BoolVar(&unused, "unused", false, "Only payees no transaction references")
	return command
}

// usedPayees collects the payee IDs the transactions and their lines
// reference.
func usedPayees(transactions []ynab.Transaction) map[string]bool {
	used := map[string]bool{}
	for _, tx := range transactions {
		if tx.PayeeID != nil {
			used[*tx.PayeeID] = true
		}
		for _, line := range tx.Subtransactions {
			if line.PayeeID != nil {
				used[*line.PayeeID] = true
			}
		}
	}
	return used
}
