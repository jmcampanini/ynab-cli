package cli

import (
	"github.com/spf13/cobra"
)

func newCategories(a *app) *cobra.Command {
	command := &cobra.Command{
		Use: "categories", Short: "Read, create, and update the plan's categories",
		Long: `Read the categories of the configured plan with the assigned, activity,
and available amounts of one month and each category's target, create
one, or change a category's name, note, group, or target. A bare
'ynab categories' prints this help and exits 0. Assigned amounts are
written by 'ynab months assign' and its siblings. The API cannot delete,
hide, or reorder categories.

` + writeHelp + "\n\n" + planHelp + "\n\n" + monthHelp,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error { return cmd.Help() }),
	}
	command.AddCommand(newCategoriesList(a), newCategoriesGet(a), newCategoriesCreate(a), newCategoriesUpdate(a))
	return command
}
