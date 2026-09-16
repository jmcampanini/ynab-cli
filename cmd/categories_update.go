package cmd

import (
	"fmt"
	"slices"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newCategoriesUpdate(a *app) *cobra.Command {
	var output outputFlags
	var target targetFlags
	var name, groupText, note string
	var dryRun bool
	command := &cobra.Command{
		Use: "update CATEGORY [--name N] [--note TEXT] [--group GROUP] [target flags]", Short: "Change a category's name, note, group, or target",
		Long: `Change fields on one category and print it as stored, in the shape of
'categories get', with the current month's amounts. CATEGORY is an ID,
an exact name, or "Group: Name", matched case-insensitively; hidden
categories match too, internal ones are refused. At least one field flag
is required; fields not named keep their values. --name renames; a name
another category of the group already has, ignoring case, is refused.
--note sets the note and an empty value clears it. --group moves the
category to another group by ID or exact name; the groups the API owns
are refused. --no-target removes the target and excludes the other target
flags. Three requests: the plans endpoint, the categories endpoint, then
the update. The API cannot delete or hide a category.

` + targetHelp + "\n\n" + writeHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab categories update Groceries --target 600 --dry-run
  ynab categories update "Bills: Internet" --name Fiber --note "" --allow-writes
  ynab categories update Vacation --target 1500 --target-date 2027-06
  ynab categories update "Old Hobby" --no-target --group Archive`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			fields := append([]string{"name", "note", "group", "no-target"}, targetFieldFlags...)
			if !slices.ContainsFunc(fields, flags.Changed) {
				return usageError("give at least one field flag to change")
			}
			if flags.Changed("name") && name == "" {
				return usageError("--name must not be empty")
			}
			if err := target.validate(cmd, false); err != nil {
				return err
			}

			s, err := a.openWrite(cmd, dryRun)
			if err != nil {
				return err
			}
			if s.categories, err = s.client.Categories(cmd.Context(), s.plan.ID); err != nil {
				return err
			}
			category, group, err := findCategory(args[0], s.categories)
			if err != nil {
				return err
			}
			if category.Internal {
				return fmt.Errorf("category %q is internal; the API does not change it", args[0])
			}
			creditCard := group.Internal && group.Name == creditCardPaymentsGroup
			var save ynab.SaveCategory
			if flags.Changed("group") {
				if group, err = s.writableGroup(groupText); err != nil {
					return err
				}
				save.GroupID = group.ID
			}
			if flags.Changed("name") {
				save.Name = name
			}
			newName := category.Name
			if save.Name != "" {
				newName = save.Name
			}
			if err := checkCategoryName(newName, group, category.ID); err != nil {
				return err
			}
			if flags.Changed("note") {
				save.Note = &note
			}
			if err := target.resolve(cmd, s, &save, creditCard); err != nil {
				return err
			}

			rest := fmt.Sprintf(" category %q", group.Name+": "+category.Name)
			after := previewCategory(category, save, group, creditCard)
			if !s.dryRun {
				if after, err = s.client.UpdateCategory(cmd.Context(), s.plan.ID, category.ID, save); err != nil {
					return err
				}
			}
			return s.printCategory(cmd, output, after, group, s.line("updated", "update", rest))
		}),
	}
	output.bind(command, false)
	command.Flags().StringVar(&name, "name", "", "New name")
	command.Flags().StringVar(&note, "note", "", "Note text; an empty value clears it")
	command.Flags().StringVar(&groupText, "group", "", "Move to this category group, by ID or exact name")
	target.bind(command, true)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print the category as it would be without changing it")
	command.ValidArgsFunction = a.completeCategories()
	completeFlags(command, a.completeCategoryGroups(), "group")
	return command
}
