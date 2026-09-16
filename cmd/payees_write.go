package cmd

import (
	"fmt"
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// maxPayeeNameLength is the API's limit on a payee name.
const maxPayeeNameLength = 500

// payeeWriteRecord is a payee record printed by a mutating command,
// marked when it is a dry-run preview.
type payeeWriteRecord struct {
	payeeRecord
	DryRun bool `json:"dry_run,omitempty"`
}

// checkPayeeName rejects a name another payee already has, ignoring
// case. exceptID is the payee being renamed.
func checkPayeeName(name string, payees []ynab.Payee, exceptID string) error {
	for _, payee := range payees {
		if payee.ID != exceptID && strings.EqualFold(payee.Name, name) {
			return fmt.Errorf("a payee named %q already exists (%s)", payee.Name, payee.ID)
		}
	}
	return nil
}

// printPayee writes one payee as a field listing, or one JSONL object,
// followed in human output by the summary line. The payees written here
// are never transfer payees, so no account names are needed.
func (s *writeSession) printPayee(cmd *cobra.Command, output outputFlags, payee ynab.Payee, summary string) error {
	record := payeeWriteRecord{payeeRecord: newPayeeRecord(payee, nil), DryRun: s.dryRun}
	out := cmd.OutOrStdout()
	if output.jsonl {
		return writeJSONL(out, []payeeWriteRecord{record})
	}

	if err := writeFields(out, [][2]string{{"id", record.ID}, {"name", record.Name}}); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "\n%s\n", summary)
	return err
}
