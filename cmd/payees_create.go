package cmd

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newPayeesCreate(a *app) *cobra.Command {
	var output outputFlags
	var dryRun bool
	command := &cobra.Command{
		Use: "create NAME", Short: "Create a payee",
		Long: `Create a payee named NAME and print it as stored, in the shape of
'payees list'. NAME is at most 500 characters, and a name another payee
already has, ignoring case, is refused, since the CLI could not then
tell them apart. 'transactions create --payee' with a new name also
creates a payee. Three requests: the plans endpoint, the payees endpoint
for the name check, then the create. The API cannot delete or merge a
payee afterwards.

` + writeHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab payees create "Corner Store" --dry-run
  ynab payees create "City Water" --allow-writes`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			if err := checkNameLength(args[0], maxPayeeNameLength); err != nil {
				return err
			}

			s, err := a.openWrite(cmd, dryRun)
			if err != nil {
				return err
			}
			payees, err := s.client.Payees(cmd.Context(), s.plan.ID)
			if err != nil {
				return err
			}
			if err := checkPayeeName(args[0], payees, ""); err != nil {
				return err
			}

			rest := fmt.Sprintf(" payee %q", args[0])
			payee := ynab.Payee{Name: args[0]}
			if !s.dryRun {
				if payee, err = s.client.CreatePayee(cmd.Context(), s.plan.ID, args[0]); err != nil {
					return err
				}
			}
			return s.printPayee(cmd, output, payee, s.line("created", "create", rest))
		}),
	}
	output.bind(command, false)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be created without creating it")
	return command
}
