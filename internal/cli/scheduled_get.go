package cli

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/selection"
	"github.com/spf13/cobra"
)

func newScheduledGet(a *app) *cobra.Command {
	var output outputFlags
	command := &cobra.Command{
		Use: "get ID", Short: "Show every field of one scheduled transaction",
		Long: `Show one scheduled transaction of the configured plan by its ID, with
every field the API carries and, for a split, its lines in a table
beneath. Three requests: the plans endpoint, the accounts endpoint (which
names transfer targets), then the scheduled transaction. --jsonl prints
one object.

` + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab scheduled get 7a3e...
  ynab scheduled get 7a3e... --jsonl`,
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
			scheduled, err := client.ScheduledTransaction(cmd.Context(), plan.ID, args[0])
			if err != nil {
				return err
			}

			record := newScheduledRecord(scheduled, selection.NewAccountNames(accounts))
			out := cmd.OutOrStdout()
			if output.jsonl {
				return render.JSONL(out, []scheduledRecord{record})
			}
			currency := plan.CurrencyFormat
			err = render.Fields(out, [][2]string{
				{"id", record.ID},
				{"next date", record.NextDate},
				{"first date", record.FirstDate},
				{"frequency", record.Frequency},
				{"account", record.Account},
				{"payee", record.Payee},
				{"category", record.Category},
				{"memo", record.Memo},
				{"amount", record.Amount.Format(currency)},
				{"flag", render.FlagWords(record.FlagColor, record.FlagName)},
				{"transfer account", record.TransferAccount},
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
		}),
	}
	output.bind(command, false)
	return command
}
