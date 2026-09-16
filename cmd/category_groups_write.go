package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

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

// checkNameLength rejects an empty name or one past the API's limit,
// counted in characters, as a usage error before any request.
func checkNameLength(name string, limit int) error {
	if name == "" {
		return usageError("NAME must not be empty")
	}
	if count := utf8.RuneCountInString(name); count > limit {
		return usageError(fmt.Sprintf("NAME is %d characters; the API allows at most %d", count, limit))
	}
	return nil
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
		return writeJSONL(out, []categoryGroupWriteRecord{record})
	}

	err := writeFields(out, [][2]string{
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
