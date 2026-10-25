// Package status classifies command errors by process exit status.
package status

import (
	"errors"
	"fmt"
)

// Exit statuses. Usage covers everything Cobra rejects before a runner
// starts; every runner failure is a command failure unless it carries
// another status.
const (
	Success            = 0
	Failure            = 1
	UsageCode          = 2
	WritesDisabledCode = 3
)

// exitError carries the process exit status for a failure.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }

func (e *exitError) Unwrap() error { return e.err }

// Usage marks a failure raised by a runner that is still a usage
// mistake, such as an empty --config value.
func Usage(message string) error {
	return &exitError{code: UsageCode, err: errors.New(message)}
}

// WritesDisabled is the failure of a mutating command run without
// allow_writes, which exits WritesDisabledCode before any request.
func WritesDisabled(configPath string) error {
	return &exitError{code: WritesDisabledCode, err: fmt.Errorf("writes are disabled: set allow_writes = true in %s or YNAB_ALLOW_WRITES=true, or pass --allow-writes; --dry-run previews without writing", configPath)}
}

// Code maps an error returned by the root command to a process exit
// status. Errors without a status are Cobra's own, raised before any
// runner started.
func Code(err error) int {
	if err == nil {
		return Success
	}
	var exit *exitError
	if errors.As(err, &exit) {
		return exit.code
	}
	return UsageCode
}

// CommandFailure gives runner failures a status unless they already carry one.
func CommandFailure(err error) error {
	if err == nil {
		return nil
	}
	var exit *exitError
	if errors.As(err, &exit) {
		return err
	}
	return &exitError{code: Failure, err: err}
}
