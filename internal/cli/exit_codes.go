package cli

import "github.com/spf13/cobra"

func exitCodesTopic() *cobra.Command {
	return helpTopic("exit-codes", "Exit codes and error categories", `ynab uses four exit statuses:

  0  Success, including an empty list, help, version, completion, and a
     dry run that previews a valid write without applying it. Reaching
     a selected read limit also succeeds. A [future-refill-funding] or
     [split-delete-recalculated] warning alone does not change success.
  1  Command failure: an API or output error; a missing or invalid
     configuration; an unmatched plan, account, category, group, payee,
     or transaction; or a value rejected after flag parsing. Examples
     include an invalid date, an amount with separators or too many
     decimals, a name past the API's length limit, a non-JSON api body,
     or --split combined with --category. A refused write, failed bulk
     update batch, or partially applied money move also exits 1.
  2  Usage: an unknown command, flag, or fixed-set operand; an operand
     count rejected by the command grammar; an unparseable flag value
     such as --color bold; or missing or conflicting flags rejected by
     the command parser. No request was made and stdout stays empty.
  3  Writes disabled: a mutating command, or api post, patch, put, or
     delete, ran without allow_writes in the configuration,
     YNAB_ALLOW_WRITES, or --allow-writes, and without --dry-run. The
     configuration was loaded, no request was made, and stdout is empty.

Errors are written as one line on stderr in the form 'ynab: <message>',
without usage text or a machine-readable error object. A raw api command
with a non-2xx response leaves stdout empty. A failed record read emits
no records. A mutating command may have printed applied records before
a later write fails; those records stay on stdout.`)
}
