package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newHelp() *cobra.Command {
	return &cobra.Command{
		Use: "help [COMMAND...]", Short: "Read help for a command or topic",
		Long: `Read command or topic help on stdout. With no command, print root help.
Unknown commands fail with a diagnostic on stderr and exit status 2.
This command does not load configuration or make API requests. It never
prompts or requires a terminal and has no machine-readable output mode.`,
		Example: "  ynab help transactions list\n  ynab help output-formats",
		Args: func(cmd *cobra.Command, args []string) error {
			_, err := findHelp(cmd.Root(), args)
			return err
		},
		RunE: run(func(cmd *cobra.Command, args []string) error {
			target, err := findHelp(cmd.Root(), args)
			if err != nil {
				return err
			}
			target.InitDefaultHelpFlag()
			return target.Help()
		}),
	}
}

func findHelp(root *cobra.Command, args []string) (*cobra.Command, error) {
	target, remaining, err := root.Find(args)
	if err != nil {
		return nil, err
	}
	if len(remaining) != 0 {
		return nil, fmt.Errorf("unknown command %q for %q; see '%s --help'", remaining[0], target.CommandPath(), target.CommandPath())
	}
	return target, nil
}
