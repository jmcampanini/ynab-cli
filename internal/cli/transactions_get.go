package cli

import (
	"github.com/jmcampanini/ynab-cli/internal/selection"
	"github.com/spf13/cobra"
)

func newTransactionsGet(a *app) *cobra.Command {
	var output outputFlags
	command := &cobra.Command{
		Use: "get ID", Short: "Show every field of one transaction",
		Long: `Show one transaction of the configured plan by its ID, with every field
the API carries and, for a split, its lines in a table beneath. A
transfer names the linked account and the transaction on its other side.
The account, payee, category, and transfer account ids appear only in
--jsonl, which prints one object with the lines in a subtransactions
array. Three requests: the plans endpoint, the accounts endpoint (which
names transfer targets), then the transaction.

` + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions get 7a3e...
  ynab transactions get 7a3e... --jsonl | jq .subtransactions`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
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
			tx, err := client.Transaction(cmd.Context(), plan.ID, args[0])
			if err != nil {
				return err
			}

			record := newTransactionRecord(tx, selection.NewAccountNames(accounts))
			out := cmd.OutOrStdout()
			if output.machine() {
				return writeRecords(out, output, []transactionRecord{record})
			}
			return writeTransactionFields(out, record, plan.CurrencyFormat)
		}),
	}
	output.bindRead(command, false)
	return command
}
