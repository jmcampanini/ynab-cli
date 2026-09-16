package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPayeesRename(a *app) *cobra.Command {
	var output outputFlags
	var dryRun bool
	command := &cobra.Command{
		Use: "rename PAYEE NAME", Short: "Rename a payee",
		Long: `Rename one payee and print it as stored, in the shape of 'payees list'.
PAYEE is a payee ID or its exact name, matched case-insensitively. A
transfer payee is refused, since its name follows its account. NAME is
at most 500 characters, and a name another payee already has, ignoring
case, is refused. Three requests: the plans endpoint, the payees
endpoint, then the update. Renaming does not merge payees; the API
cannot merge them.

` + writeHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab payees rename "AMZN Mktp" Amazon --dry-run
  ynab payees rename 7a3e... "Corner Store" --allow-writes`,
		Args: cobra.ExactArgs(2),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			if err := checkNameLength(args[1], maxPayeeNameLength); err != nil {
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
			payee, err := findPayee(args[0], payees)
			if err != nil {
				return err
			}
			if payee.TransferAccountID != nil {
				return fmt.Errorf("payee %q is a transfer payee; its name follows its account", payee.Name)
			}
			if err := checkPayeeName(args[1], payees, payee.ID); err != nil {
				return err
			}

			rest := fmt.Sprintf(" payee %q to %q", payee.Name, args[1])
			renamed := payee
			renamed.Name = args[1]
			if !s.dryRun {
				if renamed, err = s.client.UpdatePayee(cmd.Context(), s.plan.ID, payee.ID, args[1]); err != nil {
					return err
				}
			}
			return s.printPayee(cmd, output, renamed, s.line("renamed", "rename", rest))
		}),
	}
	output.bind(command, false)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print the payee as it would be without renaming it")
	return command
}
