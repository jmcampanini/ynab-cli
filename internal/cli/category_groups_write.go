package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// maxCategoryGroupNameLength is the API's limit on a group name.
const maxCategoryGroupNameLength = 50

// categoryGroupWriteRecord is a category group record printed by a
// mutating command, marked when it is a dry-run preview.
type categoryGroupWriteRecord struct {
	categoryGroupRecord
	DryRun bool `json:"dry_run,omitempty"`
}

// checkGroupName rejects a name another group already has, ignoring
// case. exceptID is the group being renamed.
func checkGroupName(name string, groups []ynab.CategoryGroup, exceptID string) error {
	for _, group := range groups {
		if group.ID != exceptID && strings.EqualFold(group.Name, name) {
			return fmt.Errorf("a category group named %q already exists (%s)", group.Name, group.ID)
		}
	}
	return nil
}

// printCategoryGroup writes one group as a field listing, or one JSONL
// object, followed in human output by the summary line.
func (s *writeSession) printCategoryGroup(cmd *cobra.Command, output outputFlags, group ynab.CategoryGroup, summary string) error {
	record := categoryGroupWriteRecord{categoryGroupRecord: newCategoryGroupRecord(group, false), DryRun: s.dryRun}
	out := cmd.OutOrStdout()
	if output.jsonl {
		return render.JSONL(out, []categoryGroupWriteRecord{record})
	}

	err := render.Fields(out, [][2]string{
		{"id", record.ID},
		{"name", record.Name},
		{"hidden", strconv.FormatBool(record.Hidden)},
		{"internal", strconv.FormatBool(record.Internal)},
		{"categories", strconv.Itoa(record.CategoryCount)},
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "\n%s\n", summary)
	return err
}
