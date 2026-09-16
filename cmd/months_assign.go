package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMonthsAssign(a *app) *cobra.Command {
	var output outputFlags
	var monthText string
	var dryRun, add, subtract bool
	command := &cobra.Command{
		Use: "assign CATEGORY AMOUNT [--add | --subtract]", Short: "Set, raise, or lower a category's assigned amount",
		Long: `Set the assigned amount of one category for one month and print the
category's row as stored, in the shape of 'categories list'. Without a
mode, AMOUNT becomes the assigned amount and may be negative, as the app
allows; a negative AMOUNT must follow "--" so it is not read as a flag.
--add and --subtract take a positive AMOUNT and write the current
assigned amount plus or minus it. No check is made against ready to
assign: this is the direct verb, and 'months move', 'cover', and 'fund'
carry the checks. Four requests: the plans endpoint, the categories
endpoint, the month, then the write.

` + monthWriteHelp + "\n\n" + writeHelp + "\n\n" + planHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab months assign Groceries 450 --dry-run
  ynab months assign "Bills: Internet" 20 --add --month 2026-10
  ynab months assign Groceries 0 --allow-writes
  ynab months assign --allow-writes Groceries -- -20`,
		Args: cobra.ExactArgs(2),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			ms, err := a.openMonth(cmd, dryRun, monthText)
			if err != nil {
				return err
			}
			amount, err := ms.amount(args[1])
			if err != nil {
				return err
			}
			if (add || subtract) && amount <= 0 {
				return usageError("--add and --subtract take a positive amount")
			}
			target, err := ms.category(args[0])
			if err != nil {
				return err
			}

			assigned := amount
			switch {
			case add:
				assigned = target.category.Assigned + amount
			case subtract:
				assigned = target.category.Assigned - amount
			}
			rest := fmt.Sprintf(" %s to %s for %s", assigned.Format(ms.plan.CurrencyFormat), target.qualifiedName(), ms.shortMonth())
			return ms.apply(cmd, output, []assignment{{assigned: assigned, before: target}}, ms.line("assigned", "assign", rest))
		}),
	}
	output.bind(command, false)
	bindMonthFlag(command, &monthText)
	command.Flags().BoolVar(&add, "add", false, "Add AMOUNT to the assigned amount")
	command.Flags().BoolVar(&subtract, "subtract", false, "Subtract AMOUNT from the assigned amount")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print the row as it would be without writing")
	command.MarkFlagsMutuallyExclusive("add", "subtract")
	command.ValidArgsFunction = firstOperand(a.completeCategories())
	return command
}
