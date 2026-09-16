package cmd

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newCategoryGroupsCreate(a *app) *cobra.Command {
	var output outputFlags
	var dryRun bool
	command := &cobra.Command{
		Use: "create NAME", Short: "Create an empty category group",
		Long: `Create a category group named NAME and print it as stored, in the shape
of 'category-groups list'. NAME is at most 50 characters, and a name
another group already has, ignoring case, is refused, since the CLI
could not then tell them apart. Three requests: the plans endpoint, the
categories endpoint for the name check, then the create. The API cannot
delete or hide a group afterwards.

` + writeHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab category-groups create Savings --dry-run
  ynab category-groups create "Annual Bills" --allow-writes`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			if err := checkNameLength(args[0], maxCategoryGroupNameLength); err != nil {
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
			if err := checkGroupName(args[0], groups, ""); err != nil {
				return err
			}

			rest := fmt.Sprintf(" category group %q", args[0])
			group := ynab.CategoryGroup{Name: args[0]}
			if !s.dryRun {
				if group, err = s.client.CreateCategoryGroup(cmd.Context(), s.plan.ID, args[0]); err != nil {
					return err
				}
			}
			return s.printCategoryGroup(cmd, output, group, s.line("created", "create", rest))
		}),
	}
	output.bind(command, false)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be created without creating it")
	return command
}
