package cmd

import (
	"fmt"
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newMonthsMove(a *app) *cobra.Command {
	var output outputFlags
	var monthText, from, to string
	var dryRun, force bool
	command := &cobra.Command{
		Use: "move AMOUNT --from CATEGORY|ready-to-assign --to CATEGORY|ready-to-assign", Short: "Move assigned money between categories",
		Long: `Move a positive AMOUNT from one category to another in one month and
print both rows as stored, in the shape of 'categories list'. The move
is two writes: the source's assigned amount less AMOUNT, then the
target's plus AMOUNT. With ready-to-assign as the source or the target,
only the other side is written, since ready to assign is what is left
unassigned. The move is refused when the source has less available than
AMOUNT, or when ready to assign is below it, unless --force is given.
Three reads then one or two writes.

` + monthWriteHelp + "\n\n" + writeHelp + "\n\n" + planHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab months move 50 --from "Dining Out" --to Groceries --dry-run
  ynab months move 200 --from ready-to-assign --to "Bills: Rent"
  ynab months move 25 --from Groceries --to ready-to-assign --allow-writes`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(cmd *cobra.Command, args []string) error {
			if strings.EqualFold(from, readyToAssignOperand) && strings.EqualFold(to, readyToAssignOperand) {
				return usageError("--from and --to are both ready-to-assign; nothing would move")
			}

			ms, err := a.openMonth(cmd, dryRun, monthText)
			if err != nil {
				return err
			}
			amount, err := ms.amount(args[0])
			if err != nil {
				return err
			}
			if amount <= 0 {
				return usageError("AMOUNT must be positive")
			}
			source, target, err := ms.endpoints(from, to)
			if err != nil {
				return err
			}
			if err := ms.checkAvailable(source, amount, force); err != nil {
				return err
			}

			rest := fmt.Sprintf(" %s from %s to %s in %s", amount.Format(ms.plan.CurrencyFormat), source.name(), target.name(), ms.shortMonth())
			return ms.apply(cmd, output, ms.transfer(source, target, amount), ms.line("moved", "move", rest))
		}),
	}
	output.bind(command, true)
	bindMonthFlag(command, &monthText)
	command.Flags().StringVar(&from, "from", "", "Source category or ready-to-assign")
	command.Flags().StringVar(&to, "to", "", "Target category or ready-to-assign")
	command.Flags().BoolVar(&force, "force", false, "Move more than the source has available")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print the rows as they would be without writing")
	_ = command.MarkFlagRequired("from")
	_ = command.MarkFlagRequired("to")
	completeFlags(command, a.completeSources(), "from", "to")
	return command
}

// endpoints resolves the two sides of a move and rejects a move from a
// category to itself; both sides ready-to-assign was rejected before any
// read.
func (ms *monthSession) endpoints(from, to string) (source, target moneySource, err error) {
	if source, err = ms.source(from); err != nil {
		return source, target, err
	}
	if target, err = ms.source(to); err != nil {
		return source, target, err
	}
	if source.same(target) {
		return source, target, usageError(fmt.Sprintf("--from and --to both name %s; nothing would move", source.name()))
	}
	return source, target, nil
}

// transfer plans the writes of a move: the source's subtraction first,
// then the target's addition, each left out for ready to assign.
func (ms *monthSession) transfer(source, target moneySource, amount ynab.Amount) []assignment {
	var writes []assignment
	if !source.readyToAssign {
		writes = append(writes, assignment{assigned: source.category.category.Assigned - amount, before: source.category})
	}
	if !target.readyToAssign {
		writes = append(writes, assignment{assigned: target.category.category.Assigned + amount, before: target.category})
	}
	return writes
}
