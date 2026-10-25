package cli

import (
	"github.com/jmcampanini/ynab-cli/internal/flagvalue"
	"github.com/spf13/cobra"
)

// bindEnumFlag adds a flag whose value must be one of allowed. typeName
// is the word help shows after the flag. The flag starts unset.
func bindEnumFlag(cmd *cobra.Command, target *string, name, typeName, usage string, allowed ...string) {
	cmd.Flags().Var(flagvalue.Enum{Allowed: allowed, Name: name, Target: target, TypeName: typeName}, name, usage)
}
