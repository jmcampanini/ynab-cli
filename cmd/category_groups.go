package cmd

import (
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// categoryGroupRecord is the --jsonl and --csv shape of a category group.
type categoryGroupRecord struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Hidden        bool   `json:"hidden"`
	Internal      bool   `json:"internal"`
	CategoryCount int    `json:"category_count"`
}

// newCategoryGroupRecord counts the group's categories, leaving hidden
// ones out unless includeHidden is set, so the count matches what
// 'categories list' shows.
func newCategoryGroupRecord(group ynab.CategoryGroup, includeHidden bool) categoryGroupRecord {
	record := categoryGroupRecord{Hidden: group.Hidden, ID: group.ID, Internal: group.Internal, Name: group.Name}
	for _, category := range group.Categories {
		if includeHidden || (!category.Hidden && !group.Hidden) {
			record.CategoryCount++
		}
	}
	return record
}

func newCategoryGroups(a *app) *cobra.Command {
	command := &cobra.Command{
		Use: "category-groups", Short: "List, create, and rename the plan's category groups",
		Long: `Read the category groups of the configured plan, create one, or rename
one. A bare 'ynab category-groups' prints this help and exits 0. The API
cannot delete, hide, or reorder groups.

` + writeHelp + "\n\n" + planHelp,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	command.AddCommand(newCategoryGroupsList(a), newCategoryGroupsCreate(a), newCategoryGroupsRename(a))
	return command
}
