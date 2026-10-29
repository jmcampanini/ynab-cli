package cli

import (
	"errors"
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/categoryview"
	"github.com/jmcampanini/ynab-cli/internal/render"
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
one write per category, the source first.

` + futureRefillHelp + "\n\n" + monthWriteHelp + "\n\n" + writeHelp + "\n\n" + planHelp + "\n\n" + monthHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab months fund --all-underfunded --dry-run
  ynab months fund Groceries "Bills: Internet" --allow-writes
  ynab months fund --all-underfunded --from "Dining Out" --month 2026-10`,
		Args: cobra.ArbitraryArgs,
		RunE: run(func(cmd *cobra.Command, args []string) error {
			switch {
			case len(args) > 0 && allUnderfunded:
				return errors.New("give categories or --all-underfunded, not both")
			case len(args) == 0 && !allUnderfunded:
				return errors.New("give at least one category or --all-underfunded")
			}

			ms, err := a.openMonth(cmd, dryRun, monthText)
			if err != nil {
				return err
			}
			candidates, err := ms.fundingCategories(args)
			if err != nil {
				return err
			}
			currency := ms.plan.CurrencyFormat
			var targets []monthCategory
			var skipped int
			for _, candidate := range candidates {
				a.warnFutureRefill(cmd, []categoryview.Record{categoryview.NewRecord(candidate.category, candidate.group, ms.apiMonth)}, currency)
				if candidate.category.TargetUnderfunded != nil && *candidate.category.TargetUnderfunded > 0 {
					targets = append(targets, candidate)
				} else if len(args) > 0 {
					skipped++
				}
			}
			skippedText := ""
			if skipped > 0 {
				skippedText = fmt.Sprintf(", %s already funded", render.CountNoun(skipped, "category", "categories"))
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
						return fmt.Errorf("--from names %s, one of the categories to fund", target.qualifiedName())
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
			rest := fmt.Sprintf(" %s with %s from %s in %s%s", render.CountNoun(len(targets), "category", "categories"), total.Format(currency), source.name(), ms.shortMonth(), skippedText)
			return ms.apply(cmd, output, writes, render.WriteSummary(ms.dryRun, "funded", "fund", rest))
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

// fundingCategories selects named categories with targets, or every visible
// category for the sweep, before filtering by API underfunding. A category
// named twice counts once, including when both its ID and name are given.
func (ms *monthSession) fundingCategories(queries []string) ([]monthCategory, error) {
	var targets []monthCategory
	seen := map[string]bool{}
	for _, query := range queries {
		target, err := ms.category(query)
		if err != nil {
			return nil, err
		}
		if seen[target.category.ID] {
			continue
		}
		seen[target.category.ID] = true
		if target.category.TargetType == nil {
			return nil, fmt.Errorf("category %q has no target to fund", query)
		}
		targets = append(targets, target)
	}
	if len(queries) > 0 {
		return targets, nil
	}

	for _, group := range ms.categories {
		if group.Hidden {
			continue
		}
		for _, category := range group.Categories {
			row, ok := ms.rows[category.ID]
			if category.Hidden || category.Internal || !ok {
				continue
			}
			targets = append(targets, monthCategory{category: row, group: group})
		}
	}
	return targets, nil
}
