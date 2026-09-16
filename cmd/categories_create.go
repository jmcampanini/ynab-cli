package cmd

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newCategoriesCreate(a *app) *cobra.Command {
	var output outputFlags
	var target targetFlags
	var groupText, note string
	var dryRun bool
	command := &cobra.Command{
		Use: "create NAME --group GROUP [--note TEXT] [target flags]", Short: "Create a category in a group",
		Long: `Create a category named NAME in GROUP and print it as stored, in the
shape of 'categories get', with the current month's amounts. GROUP is a
category group ID or its exact name, matched case-insensitively; an
internal group, such as Credit Card Payments, is refused, as the API
states. A name another category of the group already has, ignoring
case, is refused, since the CLI could not then tell them apart. --note
sets the note. The target flags need --target. Three requests: the
plans endpoint, the categories endpoint, then the create. The API cannot
delete a category afterwards; it can hide one only in the app.

` + targetHelp + "\n\n" + writeHelp + "\n\n" + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab categories create Water --group Bills --dry-run
  ynab categories create Vacation --group Savings --target 1200 \
    --target-date 2027-06 --allow-writes
  ynab categories create Groceries --group Everyday --target 600 \
    --target-frequency monthly --needs-whole-amount=false`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			if args[0] == "" {
				return usageError("NAME must not be empty")
			}
			if err := target.validate(cmd, true); err != nil {
				return err
			}

			s, err := a.openWrite(cmd, dryRun)
			if err != nil {
				return err
			}
			if s.categories, err = s.client.Categories(cmd.Context(), s.plan.ID); err != nil {
				return err
			}
			group, err := s.writableGroup(groupText)
			if err != nil {
				return err
			}
			if err := checkCategoryName(args[0], group, ""); err != nil {
				return err
			}
			save := ynab.SaveCategory{GroupID: group.ID, Name: args[0]}
			if cmd.Flags().Changed("note") {
				save.Note = &note
			}
			if err := target.resolve(cmd, s, &save, false); err != nil {
				return err
			}

			rest := fmt.Sprintf(" category %q", group.Name+": "+args[0])
			if s.dryRun {
				return s.printCategory(cmd, output, previewCategory(ynab.Category{}, save, group, false), group, s.line("created", "create", rest))
			}
			stored, err := s.client.CreateCategory(cmd.Context(), s.plan.ID, save)
			if err != nil {
				return err
			}
			return s.printCategory(cmd, output, stored, group, s.line("created", "create", rest))
		}),
	}
	output.bind(command, false)
	command.Flags().StringVar(&groupText, "group", "", "Category group ID or exact name")
	command.Flags().StringVar(&note, "note", "", "Note text")
	target.bind(command, false)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be created without creating it")
	_ = command.MarkFlagRequired("group")
	return command
}
