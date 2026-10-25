// Package flagvalue validates values accepted by command flags.
package flagvalue

import (
	"fmt"
	"slices"
	"strings"
)

// Enum binds a flag to one of a fixed set of words and rejects any
// other value as a usage error before the runner starts.
type Enum struct {
	Allowed  []string
	Name     string
	Target   *string
	TypeName string
}

// Set accepts an allowed value and leaves the target unchanged on error.
func (v Enum) Set(value string) error {
	if !slices.Contains(v.Allowed, value) {
		return fmt.Errorf("invalid --%s value %q: must be %s", v.Name, value, strings.Join(v.Allowed, ", "))
	}
	*v.Target = value
	return nil
}

// String returns the current flag value.
func (v Enum) String() string { return *v.Target }

// Type returns the value label shown in flag help.
func (v Enum) Type() string { return v.TypeName }
