package cli

import (
	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/selection"
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
			plan, err := selection.Plan(cmd.Context(), client, loaded.Config)
			if err != nil {
				return err
			}
			accounts, err := client.Accounts(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}
			account, err := selection.Account(args[0], accounts)
			if err != nil {
				return err
			}

			record := newAccountRecord(account)
			out := cmd.OutOrStdout()
			if output.machine() {
				return writeRecords(out, output, []accountRecord{record})
			}
			return render.Fields(out, accountFields(record, plan.CurrencyFormat))
		}),
	}
	output.bindRead(command, false)
	command.ValidArgsFunction = a.completeAccounts()
	return command
}
