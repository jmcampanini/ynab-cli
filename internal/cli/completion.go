package cli

import "github.com/spf13/cobra"

func newCompletion() *cobra.Command {
	return &cobra.Command{
		Use: "completion SHELL", Short: "Print a shell completion script",
		Long: `Print a completion script for bash, zsh, fish, or powershell on stdout.
Diagnostics go to stderr. Load the script through your shell's completion
setup. This command does not create files, load configuration, or make
API requests. It never prompts or requires a terminal and has no JSON,
JSONL, or CSV output mode.

Once installed, name completion reads your configuration and queries the
plan for account, category, category group, and payee names. If a lookup
fails, completion offers no names.`,
		Example:   "  ynab completion zsh",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			root, out := cmd.Root(), cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(out, true)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			default:
				return root.GenPowerShellCompletionWithDesc(out)
			}
		}),
	}
}
