package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// importRecord is the --jsonl shape of an import.
type importRecord struct {
	Count          int      `json:"count"`
	TransactionIDs []string `json:"transaction_ids"`
}

func newTransactionsImport(a *app) *cobra.Command {
	var output outputFlags
	var dryRun bool
	command := &cobra.Command{
		Use: "import", Short: "Import transactions from the linked accounts",
		Long: `Ask YNAB to import from every account linked to a bank, as the app's
import button does, and print how many transactions arrived; --jsonl
prints one object with the count and the transaction ids. Exit 0 whether
that is zero or more. Nothing is imported for accounts without a link or
in import error; 'accounts list' shows the link state. Three requests:
the plans endpoint, the accounts endpoint, then the import. A dry run
prints only the summary line, since the API offers no preview, and
prints nothing with --jsonl.

` + writeHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab transactions import --allow-writes
  ynab transactions import --allow-writes --jsonl | jq .count`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			s, err := a.openWrite(cmd, dryRun)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if s.dryRun {
				if output.jsonl {
					return nil
				}
				_, err := fmt.Fprintln(out, "dry run, nothing changed: would import from the linked accounts")
				return err
			}
			ids, err := s.client.ImportTransactions(cmd.Context(), s.plan.ID)
			if err != nil {
				return err
			}
			if ids == nil {
				ids = []string{}
			}
			if output.jsonl {
				return writeJSONL(out, []importRecord{{Count: len(ids), TransactionIDs: ids}})
			}
			_, err = fmt.Fprintf(out, "imported %s\n", transactionCount(len(ids)))
			return err
		}),
	}
	output.bind(command, false)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print the summary line without importing")
	return command
}
