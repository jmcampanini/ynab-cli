package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

// Exit statuses. Usage covers everything Cobra rejects before a runner
// starts; every runner failure is a command failure unless it carries
// another status.
const (
	ExitSuccess        = 0
	ExitFailure        = 1
	ExitUsage          = 2
	ExitWritesDisabled = 3
)

// exitError carries the process exit status for a failure.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }

func (e *exitError) Unwrap() error { return e.err }

// usageError marks a failure raised by a runner that is still a usage
// mistake, such as an empty --config value.
func usageError(message string) error {
	return &exitError{code: ExitUsage, err: errors.New(message)}
}

// ExitCode maps an error returned by the root command to a process exit
// status. Errors without a status are Cobra's own, raised before any
// runner started.
func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var exit *exitError
	if errors.As(err, &exit) {
		return exit.code
	}
	return ExitUsage
}

// run wraps a leaf runner so its failures exit ExitFailure unless they
// already carry a status.
func run(runner func(*cobra.Command, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := runner(cmd, args)
		if err == nil {
			return nil
		}
		var exit *exitError
		if errors.As(err, &exit) {
			return err
		}
		return &exitError{code: ExitFailure, err: err}
	}
}

func exitCodesTopic() *cobra.Command {
	return &cobra.Command{
		Use: "exit-codes", Short: "Exit codes and error categories", Args: cobra.NoArgs,
		Long: `ynab uses four exit statuses:

  0  Success, including an empty list, help, version, completion, and
     'ynab help <unknown>', which prints the root usage.
  1  Command failure: the API returned an error, the token or plan is not
     configured or not found, an account, category, payee, or transaction
     did not match, or the configuration file could not be loaded.
  2  Usage: an unknown command, flag, or operand count, an invalid flag
     value such as --color bold, or a month that is not current, YYYY-MM,
     or YYYY-MM-01. Nothing ran and stdout is empty.
  3  Writes disabled: reserved for mutating commands run without
     allow_writes. No command in this release exits 3.

Errors are written to stderr as 'ynab: <message>'. Machine output that was
already written stays on stdout when a later failure changes the status.`,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
}
