package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMonthsCover(a *app) *cobra.Command {
	var output outputFlags
	var monthText, from string
	var dryRun, force bool
	command := &cobra.Command{
		Use: "cover CATEGORY --from SOURCE", Short: "Move exactly an overspent category's shortfall into it",
		Long: `Bring an overspent category's available amount back to zero by moving
exactly the shortfall from SOURCE, a category or ready-to-assign, and
print both rows as stored, in the shape of 'categories list'. When the
category's available amount is not negative, nothing is written, the
command says there is nothing to cover, and the exit status is 0. The
move is refused when SOURCE has less available than the shortfall, or
when ready to assign is below it, unless --force is given. Three reads
then one or two writes, the source first.

` + monthWriteHelp + "\n\n" + writeHelp + "\n\n" + planHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab months cover "Dining Out" --from Groceries --dry-run
  ynab months cover "Dining Out" --from ready-to-assign --allow-writes`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			ms, err := a.openMonth(cmd, dryRun, monthText)
			if err != nil {
				return err
			}
			target, err := ms.category(args[0])
			if err != nil {
				return err
			}
			currency := ms.plan.CurrencyFormat
			if target.category.Available >= 0 {
				return ms.printCategories(cmd, output, nil, fmt.Sprintf("nothing to cover: %s has %s available in %s", target.qualifiedName(), target.category.Available.Format(currency), ms.shortMonth()))
			}
			source, err := ms.source(from)
			if err != nil {
				return err
			}
			if source.same(moneySource{category: target}) {
				return usageError(fmt.Sprintf("--from names %s, the category to cover", target.qualifiedName()))
			}
			shortfall := -target.category.Available
			if err := ms.checkAvailable(source, shortfall, force); err != nil {
				return err
			}

			rest := fmt.Sprintf(" %s with %s from %s in %s", target.qualifiedName(), shortfall.Format(currency), source.name(), ms.shortMonth())
			return ms.apply(cmd, output, ms.transfer(source, moneySource{category: target}, shortfall), ms.line("covered", "cover", rest))
		}),
	}
	output.bind(command, true)
	bindMonthFlag(command, &monthText)
	command.Flags().StringVar(&from, "from", "", "Source category or ready-to-assign")
	command.Flags().BoolVar(&force, "force", false, "Cover with more than the source has available")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print the rows as they would be without writing")
	_ = command.MarkFlagRequired("from")
	return command
}
