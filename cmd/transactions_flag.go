package cmd

import (
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newTransactionsFlag(a *app) *cobra.Command {
	var output outputFlags
	var dryRun bool
	var color string
	command := &cobra.Command{
		Use: "flag --color COLOR|none ID...", Short: "Set or remove the flag on transactions",
		Long: `Set one flag color on the transactions named through one bulk update
and print the stored records in the shape of 'transactions list'.
--color takes red, orange, yellow, green, blue, purple, or none, which
removes the flag. A flag's custom name belongs to the color in the app
and cannot be set here.

Requests: the plans endpoint, the accounts endpoint, then one bulk
update per 100 transactions.

` + writeHelp + "\n\n" + bulkHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions flag --color red 7a3e... --dry-run
  ynab transactions flag --color none 7a3e... 9c1d... --allow-writes`,
		Args: cobra.MinimumNArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			s, err := a.openWrite(cmd, dryRun)
			if err != nil {
				return err
			}

			change := ynab.SaveTransaction{FlagColor: ynab.SomeValue(color)}
			past := "flagged"
			if color == "none" {
				change.FlagColor = ynab.NullValue[string]()
				past = "unflagged"
			}
			return s.bulk(cmd, output, bulkWrite{base: "flag", change: change, ids: args, past: past})
		}),
	}
	output.bind(command, true)
	bindEnumFlag(command, &color, "color", "color", "Flag color, or none to remove the flag", flagColors...)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Preview the records without changing them")
	_ = command.MarkFlagRequired("color")
	return command
}
