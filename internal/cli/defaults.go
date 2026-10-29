package cli

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/config"
	"github.com/jmcampanini/ynab-cli/internal/status"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const commandDefaultAnnotation = "command-default"

// useCommandDefault opts a flag into the operational default with its name.
// Optional filters can use the same name without inheriting that default.
func useCommandDefault(flag *pflag.Flag) {
	if flag.Annotations == nil {
		flag.Annotations = make(map[string][]string)
	}
	flag.Annotations[commandDefaultAnnotation] = []string{"true"}
}

// configureDefaults keeps help accurate without setting flag values during
// construction. The pre-run hook applies defaults after argument parsing and
// leaves explicitly supplied values and their Changed markers intact.
func configureDefaults(root *cobra.Command) {
	defaults := config.CommandDefaults()
	var document func(*cobra.Command)
	document = func(cmd *cobra.Command) {
		for name, value := range defaults {
			flag := cmd.Flags().Lookup(name)
			if flag == nil {
				flag = cmd.PersistentFlags().Lookup(name)
			}
			if flag != nil && len(flag.Annotations[commandDefaultAnnotation]) != 0 {
				flag.DefValue = value
			}
		}
		for _, child := range cmd.Commands() {
			document(child)
		}
	}
	document(root)
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		for name, value := range defaults {
			if flag := cmd.Flags().Lookup(name); flag != nil && !flag.Changed && len(flag.Annotations[commandDefaultAnnotation]) != 0 {
				if err := flag.Value.Set(value); err != nil {
					return status.CommandFailure(fmt.Errorf("default for --%s: %w", name, err))
				}
			}
		}
		return nil
	}
}
