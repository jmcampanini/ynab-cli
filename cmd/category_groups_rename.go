package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCategoryGroupsRename(a *app) *cobra.Command {
	var output outputFlags
	var dryRun bool
	command := &cobra.Command{
		Use: "rename GROUP NAME", Short: "Rename a category group",
		Long: `Rename one category group and print it as stored, in the shape of
'category-groups list'. GROUP is a group ID or its exact name, matched
case-insensitively; hidden groups match too, and the groups the API
owns, Credit Card Payments and the master group, are refused.
NAME is at most 50 characters, and a name another group already has,
ignoring case, is refused. Three requests: the plans endpoint, the
categories endpoint, then the update.

` + writeHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab category-groups rename Bills "Monthly Bills" --dry-run
  ynab category-groups rename 7a3e... Fun --allow-writes`,
		Args: cobra.ExactArgs(2),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			if err := checkNameLength(args[1], maxCategoryGroupNameLength); err != nil {
				return err
			}

			s, err := a.openWrite(cmd, dryRun)
			if err != nil {
				return err
			}
			groups, err := s.client.Categories(cmd.Context(), s.plan.ID)
			if err != nil {
				return err
			}
			group, err := findCategoryGroup(args[0], groups)
			if err != nil {
				return err
			}
			if apiOwnedGroup(group) {
				return fmt.Errorf("group %q belongs to the API; it is not renamed", group.Name)
			}
			if err := checkGroupName(args[1], groups, group.ID); err != nil {
				return err
			}

			rest := fmt.Sprintf(" category group %q to %q", group.Name, args[1])
			if s.dryRun {
				group.Name = args[1]
				return s.printCategoryGroup(cmd, output, group, s.line("renamed", "rename", rest))
			}
			stored, err := s.client.UpdateCategoryGroup(cmd.Context(), s.plan.ID, group.ID, args[1])
			if err != nil {
				return err
			}
			if stored.Categories == nil {
				// The update answer omits the categories; the count comes from the listing.
				stored.Categories = group.Categories
			}
			return s.printCategoryGroup(cmd, output, stored, s.line("renamed", "rename", rest))
		}),
	}
	output.bind(command, false)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print the group as it would be without renaming it")
	return command
}
