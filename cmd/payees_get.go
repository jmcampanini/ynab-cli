package cmd

import "github.com/spf13/cobra"

func newPayeesGet(a *app) *cobra.Command {
	var output outputFlags
	command := &cobra.Command{
		Use: "get PAYEE", Short: "Show every field of one payee",
		Long: `Show one payee of the configured plan. PAYEE is the payee ID or its exact
name, matched case-insensitively. No prefix or fuzzy matching: a miss
fails and lists the closest names. Three requests: the plans endpoint, the
accounts endpoint (which names a transfer payee's account), then the
payees endpoint. --jsonl prints one object.

` + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab payees get Costco
  ynab payees get "Transfer : Visa" --jsonl`,
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
			payees, err := client.Payees(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}
			payee, err := findPayee(args[0], payees)
			if err != nil {
				return err
			}

			record := newPayeeRecord(payee, newAccountNames(accounts))
			out := cmd.OutOrStdout()
			if output.jsonl {
				return writeJSONL(out, []payeeRecord{record})
			}
			return writeFields(out, [][2]string{
				{"id", record.ID},
				{"name", record.Name},
				{"transfer account", record.TransferAccount},
				{"transfer account id", record.TransferAccountID},
			})
		}),
	}
	output.bind(command, false)
	return command
}
