package cmd

import (
	"errors"
	"fmt"

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

// writesDisabled is the failure of a mutating command run without
// allow_writes, which exits ExitWritesDisabled before any request.
func writesDisabled(configPath string) error {
	return &exitError{code: ExitWritesDisabled, err: fmt.Errorf("writes are disabled: set allow_writes = true in %s or YNAB_ALLOW_WRITES=true, or pass --allow-writes; --dry-run previews without writing", configPath)}
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
     configured or not found, an account, category, group, payee, or
     transaction did not match, a later batch of a bulk update or a
     later write of a money move failed, a cleared state change or a move
     past the available amount needed --force, a name to create already
     exists, a target flag is not supported on the category, or the
     configuration file could not be loaded.
  2  Usage: an unknown command, flag, or operand count, an invalid flag
     value such as --color bold, a month that is not current, YYYY-MM, or
     YYYY-MM-01, an amount with separators or too many decimals, a name
     past the API's length limit, or conflicting write flags such as
     --split with --category or --target-frequency with --target-date.
     Nothing ran and stdout is empty.
  3  Writes disabled: a mutating command ran without allow_writes in the
     configuration, YNAB_ALLOW_WRITES, or --allow-writes, and without
     --dry-run. The configuration was loaded, no request was made, and
     stdout is empty.

Errors are written to stderr as 'ynab: <message>'. Machine output that was
already written stays on stdout when a later failure changes the status.`,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
}
