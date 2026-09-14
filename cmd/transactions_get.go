package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newTransactionsGet(a *app) *cobra.Command {
	var output outputFlags
	command := &cobra.Command{
		Use: "get ID", Short: "Show every field of one transaction",
		Long: `Show one transaction of the configured plan by its ID, with every field
the API carries and, for a split, its lines in a table beneath. A
transfer names the linked account and the transaction on its other side.
Three requests: the plans endpoint, the accounts endpoint (which names
transfer targets), then the transaction. --jsonl prints one object with
the lines in a subtransactions array.

` + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions get 7a3e...
  ynab transactions get 7a3e... --jsonl | jq .subtransactions`,
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
			tx, err := client.Transaction(cmd.Context(), plan.ID, args[0])
			if err != nil {
				return err
			}

			record := newTransactionRecord(tx, newAccountNames(accounts))
			out := cmd.OutOrStdout()
			if output.jsonl {
				return writeJSONL(out, []transactionRecord{record})
			}
			currency := plan.CurrencyFormat
			err = writeFields(out, [][2]string{
				{"id", record.ID},
				{"date", record.Date},
				{"account", record.Account},
				{"payee", record.Payee},
				{"category", record.Category},
				{"memo", record.Memo},
				{"amount", record.Amount.Format(currency)},
				{"cleared", record.Cleared},
				{"approved", yesNo(record.Approved)},
				{"flag", flagWords(record.FlagColor, record.FlagName)},
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
			return writeTable(out, lineColumns, lineRows(record.Subtransactions, currency))
		}),
	}
	output.bind(command, false)
	return command
}
