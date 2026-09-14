package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// enumValue binds a flag to one of a fixed set of words and rejects any
// other value as a usage error before the runner starts.
type enumValue struct {
	allowed  []string
	name     string
	target   *string
	typeName string
}

func (v enumValue) Set(value string) error {
	if !slices.Contains(v.allowed, value) {
		return fmt.Errorf("invalid --%s value %q: must be %s", v.name, value, strings.Join(v.allowed, ", "))
	}
	*v.target = value
	return nil
}

func (v enumValue) String() string { return *v.target }

func (v enumValue) Type() string { return v.typeName }

// bindEnumFlag adds a flag whose value must be one of allowed. typeName
// is the word help shows after the flag. The flag starts unset.
func bindEnumFlag(cmd *cobra.Command, target *string, name, typeName, usage string, allowed ...string) {
	cmd.Flags().Var(enumValue{allowed: allowed, name: name, target: target, typeName: typeName}, name, usage)
}
