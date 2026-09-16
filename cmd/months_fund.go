package cmd

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func newMonthsFund(a *app) *cobra.Command {
	var output outputFlags
	var monthText, from string
	var dryRun, force, allUnderfunded bool
	command := &cobra.Command{
		Use: "fund CATEGORY... | --all-underfunded [--from SOURCE]", Short: "Assign what underfunded targets still need",
		Long: `Add each category's underfunded amount, what its target still needs
this month, to its assigned amount, and print the rows changed as stored,
in the shape of 'categories list', with the total in the summary. Name
the categories, or pass --all-underfunded to fund every visible category
whose target needs money; hidden categories are then left out, as in
'plans status'. A named category without a target fails; one whose target
needs nothing is skipped and counted in the summary.

The money comes from ready to assign, and the command refuses to leave
it negative unless --force is given. --from SOURCE instead subtracts the
total from a category first, refusing when it has less available unless
--force is given, or from ready-to-assign explicitly. Three reads then
one write per category, the source first. For a "plan your spending"
target in a future month, the API's underfunded amount counts funding
from earlier periods, which the YNAB app ignores, so the two can differ.

` + monthWriteHelp + "\n\n" + writeHelp + "\n\n" + planHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab months fund --all-underfunded --dry-run
  ynab months fund Groceries "Bills: Internet" --allow-writes
  ynab months fund --all-underfunded --from "Dining Out" --month 2026-10`,
		Args: cobra.ArbitraryArgs,
		RunE: run(func(cmd *cobra.Command, args []string) error {
			switch {
			case len(args) > 0 && allUnderfunded:
				return usageError("give categories or --all-underfunded, not both")
			case len(args) == 0 && !allUnderfunded:
				return usageError("give at least one category or --all-underfunded")
			}

			ms, err := a.openMonth(cmd, dryRun, monthText)
			if err != nil {
				return err
			}
			targets, skipped, err := ms.underfunded(args)
			if err != nil {
				return err
			}
			currency := ms.plan.CurrencyFormat
			skippedText := ""
			if skipped > 0 {
				skippedText = fmt.Sprintf(", %s already funded", countNoun(skipped, "category", "categories"))
			}
			if len(targets) == 0 {
				return ms.printCategories(cmd, output, nil, fmt.Sprintf("nothing to fund in %s%s", ms.shortMonth(), skippedText))
			}

			var total ynab.Amount
			for _, target := range targets {
				total += *target.category.TargetUnderfunded
			}
			source := moneySource{readyToAssign: true}
			if cmd.Flags().Changed("from") {
				if source, err = ms.source(from); err != nil {
					return err
				}
				for _, target := range targets {
					if source.same(moneySource{category: target}) {
						return usageError(fmt.Sprintf("--from names %s, one of the categories to fund", target.qualifiedName()))
					}
				}
			}
			if err := ms.checkAvailable(source, total, force); err != nil {
				return err
			}

			var writes []assignment
			if !source.readyToAssign {
				writes = append(writes, assignment{assigned: source.category.category.Assigned - total, before: source.category})
			}
			for _, target := range targets {
				writes = append(writes, assignment{assigned: target.category.Assigned + *target.category.TargetUnderfunded, before: target})
			}
			rest := fmt.Sprintf(" %s with %s from %s in %s%s", countNoun(len(targets), "category", "categories"), total.Format(currency), source.name(), ms.shortMonth(), skippedText)
			return ms.apply(cmd, output, writes, ms.line("funded", "fund", rest))
		}),
	}
	output.bind(command, true)
	bindMonthFlag(command, &monthText)
	command.Flags().BoolVar(&allUnderfunded, "all-underfunded", false, "Fund every visible category whose target needs money")
	command.Flags().StringVar(&from, "from", "", "Take the total from this category or ready-to-assign")
	command.Flags().BoolVar(&force, "force", false, "Fund past what the source has available")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Print the rows as they would be without writing")
	command.ValidArgsFunction = a.completeCategories()
	completeFlags(command, a.completeSources(), "from")
	return command
}

// underfunded selects the categories to fund: the named ones whose
// target needs money, or with no names every visible one that does. A
// category named twice, by ID and name or in both name forms, counts
// once. skipped counts named categories whose target needs nothing.
func (ms *monthSession) underfunded(queries []string) (targets []monthCategory, skipped int, err error) {
	needsMoney := func(category ynab.Category) bool {
		return category.TargetUnderfunded != nil && *category.TargetUnderfunded > 0
	}
	seen := map[string]bool{}
	for _, query := range queries {
		target, err := ms.category(query)
		if err != nil {
			return nil, 0, err
		}
		if seen[target.category.ID] {
			continue
		}
		seen[target.category.ID] = true
		if target.category.TargetType == nil {
			return nil, 0, fmt.Errorf("category %q has no target to fund", query)
		}
		if !needsMoney(target.category) {
			skipped++
			continue
		}
		targets = append(targets, target)
	}
	if len(queries) > 0 {
		return targets, skipped, nil
	}

	for _, group := range ms.categories {
		if group.Hidden {
			continue
		}
		for _, category := range group.Categories {
			row, ok := ms.rows[category.ID]
			if category.Hidden || category.Internal || !ok || !needsMoney(row) {
				continue
			}
			targets = append(targets, monthCategory{category: row, group: group})
		}
	}
	return targets, 0, nil
}
