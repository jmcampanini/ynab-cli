package cli

import "github.com/spf13/cobra"

func exitCodesTopic() *cobra.Command {
	return helpTopic("exit-codes", "Exit codes and error categories", `ynab uses four exit statuses:

  0  Success, including an empty list, help, version, completion, and
     'ynab help <unknown>', which prints the root usage. A
     [future-refill-funding] or [split-delete-recalculated] warning alone
     does not change success.
  1  Command failure: the API returned an error, the token or plan is not
     configured or not found, an account, category, group, payee, or
     transaction did not match, a later batch of a bulk update or a
     later write of a money move or split-deletion workaround failed,
     a cleared state change or a move
     past the available amount needed --force, a name to create already
     exists, a target flag is not supported on the category, the
     configuration file could not be loaded, or an api command got a
     non-2xx response, whose body is on stdout and status on stderr.
  2  Usage: an unknown command, flag, or operand count, an invalid flag
     value such as --color bold, a month that is not current, YYYY-MM, or
     YYYY-MM-01, an amount with separators or too many decimals, a name
     past the API's length limit, an api body that is not JSON, or
     conflicting flags such as --split with --category, --month with
     --since, or --body with --body-file. Nothing ran and stdout is
     empty.
  3  Writes disabled: a mutating command, or api post, patch, put, or
     delete, ran without allow_writes in the configuration,
     YNAB_ALLOW_WRITES, or --allow-writes, and without --dry-run. The
     configuration was loaded, no request was made, and stdout is empty.

Errors are written to stderr as 'ynab: <message>'. Machine output that was
already written stays on stdout when a later failure changes the status.`)
}
