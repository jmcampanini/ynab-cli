package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmcampanini/ynab-cli/internal/names"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// Shared machinery for categories create and update: the target flags
// and the spec's rules for them, the dry-run preview, and the printing.

// targetFrequencies are the cadences the API's goal_frequency accepts.
var targetFrequencies = []string{"monthly", "weekly", "yearly"}

// targetCadences are the cadence codes a category reports for each
// frequency, as 'categories get' decodes them.
var targetCadences = map[string]int{"monthly": 1, "weekly": 2, "yearly": 13}

// targetFlags are the target flags create and update share, as typed.
type targetFlags struct {
	amount, date, frequency    string
	needsWholeAmount, noTarget bool
}

// targetFieldFlags are the target flags that set a field. create binds
// these; update binds --no-target alongside them.
var targetFieldFlags = []string{"target", "target-date", "target-frequency", "needs-whole-amount"}

func (f *targetFlags) bind(cmd *cobra.Command, update bool) {
	cmd.Flags().StringVar(&f.amount, "target", "", "Target amount in currency units")
	cmd.Flags().StringVar(&f.date, "target-date", "", "Target date, YYYY-MM-DD or YYYY-MM, for a by-date target")
	bindEnumFlag(cmd, &f.frequency, "target-frequency", "frequency", "Repeat the target: monthly, weekly, yearly", targetFrequencies...)
	cmd.Flags().BoolVar(&f.needsWholeAmount, "needs-whole-amount", false, "Set aside the full amount each period; =false refills up to it")
	cmd.MarkFlagsMutuallyExclusive("target-frequency", "target-date")
	if update {
		cmd.Flags().BoolVar(&f.noTarget, "no-target", false, "Remove the target")
		for _, name := range targetFieldFlags {
			cmd.MarkFlagsMutuallyExclusive("no-target", name)
		}
	}
}

// validate applies the rules that need no read: the date's form,
// --target-frequency requires --target, and on create so does every
// other target flag, since there is no target yet to change.
func (f *targetFlags) validate(cmd *cobra.Command, create bool) error {
	flags := cmd.Flags()
	if flags.Changed("target-date") {
		if _, err := targetDate(f.date); err != nil {
			return err
		}
	}
	if flags.Changed("target") {
		return nil
	}
	if flags.Changed("target-frequency") {
		return usageError("--target-frequency requires --target")
	}
	if create {
		for _, name := range []string{"target-date", "needs-whole-amount"} {
			if flags.Changed(name) {
				return usageError(fmt.Sprintf("--%s requires --target on create", name))
			}
		}
	}
	return nil
}

// resolve fills the target fields of a save from the flags given. On a
// credit card payment category the API supports only the amount, so
// --target-frequency and --needs-whole-amount are refused there.
func (f *targetFlags) resolve(cmd *cobra.Command, s *writeSession, save *ynab.SaveCategory, creditCard bool) error {
	flags := cmd.Flags()
	if creditCard {
		for _, name := range []string{"target-frequency", "needs-whole-amount"} {
			if flags.Changed(name) {
				return fmt.Errorf("--%s is not supported on a credit card payment category; the API allows only --target there", name)
			}
		}
	}
	if f.noTarget {
		save.TargetAmount = ynab.NullValue[int64]()
		return nil
	}

	if flags.Changed("target") {
		amount, err := s.amount(f.amount)
		if err != nil {
			return err
		}
		if amount <= 0 {
			return usageError("--target must be positive")
		}
		save.SetTargetAmount(amount)
	}
	if flags.Changed("target-date") {
		// validate has already accepted the form.
		date, _ := targetDate(f.date)
		save.TargetDate = &date
	}
	if flags.Changed("target-frequency") {
		save.TargetFrequency = f.frequency
	}
	if flags.Changed("needs-whole-amount") {
		needsWhole := f.needsWholeAmount
		save.TargetNeedsWholeAmt = &needsWhole
	}
	return nil
}

// targetDate parses YYYY-MM-DD, or YYYY-MM as the first of that month,
// since by-date targets are month-granular. A bad form is a usage error.
func targetDate(text string) (string, error) {
	if parsed, err := time.Parse(time.DateOnly, text); err == nil {
		return parsed.Format(time.DateOnly), nil
	}
	if parsed, err := time.Parse("2006-01", text); err == nil {
		return parsed.Format(time.DateOnly), nil
	}
	return "", usageError(fmt.Sprintf("invalid --target-date %q: use YYYY-MM-DD or YYYY-MM", text))
}

// previewCategory applies a save to a category as the API does, for a
// dry run: an amount on a category without a target makes a "plan your
// spending" target, or a "monthly funding" one on a credit card payment
// category; a date makes a target balance with a date; a frequency
// makes it repeat; null removes it. Computed fields such as the
// underfunded amount are left as they were.
func previewCategory(category ynab.Category, save ynab.SaveCategory, group ynab.CategoryGroup, creditCard bool) ynab.Category {
	if save.Name != "" {
		category.Name = save.Name
	}
	if save.Note != nil {
		category.Note = save.Note
	}
	category.GroupID, category.GroupName = group.ID, group.Name

	if save.TargetAmount.Set && save.TargetAmount.Value == nil {
		category.TargetType, category.TargetAmount, category.TargetDate = nil, 0, nil
		category.TargetCadence, category.TargetCadenceFreq, category.TargetNeedsWholeAmt = nil, nil, nil
		return category
	}
	if save.TargetAmount.Set {
		category.TargetAmount = ynab.Amount(*save.TargetAmount.Value)
		if category.TargetType == nil {
			targetType := "NEED"
			if creditCard {
				targetType = "MF"
			}
			category.TargetType = &targetType
		}
	}
	if save.TargetDate != nil {
		// The API answers a dated target as TB with a date; TBD is the
		// code older targets carry.
		targetType := "TB"
		category.TargetType, category.TargetDate = &targetType, save.TargetDate
	}
	if save.TargetFrequency != "" {
		targetType, cadence, every := "NEED", targetCadences[save.TargetFrequency], 1
		category.TargetType, category.TargetCadence, category.TargetCadenceFreq, category.TargetDate = &targetType, &cadence, &every, nil
	}
	if save.TargetNeedsWholeAmt != nil {
		category.TargetNeedsWholeAmt = save.TargetNeedsWholeAmt
	}
	return category
}

// printCategory writes one category as 'categories get' does, followed
// in human output by the summary line.
func (s *writeSession) printCategory(cmd *cobra.Command, output outputFlags, category ynab.Category, group ynab.CategoryGroup, summary string) error {
	apiMonth, err := s.app.month("current")
	if err != nil {
		return err
	}
	records := s.categoryWriteRecords([]categoryRecord{newCategoryRecord(category, group, apiMonth)})
	out := cmd.OutOrStdout()
	if output.jsonl {
		return writeJSONL(out, records)
	}

	if err := writeFields(out, categoryFields(records[0].categoryRecord, s.plan.CurrencyFormat)); err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "\n%s\n", summary)
	return err
}

// writableGroup resolves a --group operand and refuses a group the API
// owns, which it does not place categories in.
func (s *writeSession) writableGroup(query string) (ynab.CategoryGroup, error) {
	group, err := findCategoryGroup(query, s.categories)
	if err != nil {
		return ynab.CategoryGroup{}, err
	}
	if apiOwnedGroup(group) {
		return ynab.CategoryGroup{}, fmt.Errorf("group %q belongs to the API; it does not place categories there", group.Name)
	}
	return group, nil
}

// apiOwnedGroup reports whether the API owns the group: the master
// group, whose categories are internal, and Credit Card Payments. The
// API also marks every group it created from its plan template as
// internal, such as Bills, so the flag alone does not settle it.
func apiOwnedGroup(group ynab.CategoryGroup) bool {
	if group.Internal && group.Name == creditCardPaymentsGroup {
		return true
	}
	for _, category := range group.Categories {
		if category.Internal {
			return true
		}
	}
	return false
}

// checkCategoryName refuses a name another category of the group already
// has, ignoring case, since the CLI could not then tell them apart by
// name. exceptID is the category being renamed.
func checkCategoryName(name string, group ynab.CategoryGroup, exceptID string) error {
	for _, category := range group.Categories {
		if category.ID != exceptID && strings.EqualFold(category.Name, name) {
			return fmt.Errorf("group %q already has a category named %q (%s)", group.Name, category.Name, category.ID)
		}
	}
	return nil
}

// findCategoryGroup resolves query against group IDs, then names
// ignoring case, hidden groups included.
func findCategoryGroup(query string, groups []ynab.CategoryGroup) (ynab.CategoryGroup, error) {
	candidates := make([]names.Candidate, len(groups))
	for i, group := range groups {
		candidates[i] = names.Candidate{ID: group.ID, Names: []string{group.Name}}
	}

	index, err := names.Resolve("category group", query, candidates)
	if err != nil {
		return ynab.CategoryGroup{}, err
	}
	return groups[index], nil
}

// targetHelp is the target flags paragraph create and update share.
const targetHelp = `Targets:
  --target AMOUNT sets the target amount. Alone on a category without a
  target it makes a monthly "plan your spending" target, or a "monthly
  funding" target on a credit card payment category. With --target-date
  it makes a target balance (TB) with that date; the date is YYYY-MM-DD
  or YYYY-MM.
  With --target-frequency monthly, weekly, or yearly it makes a repeating
  "plan your spending" target, replacing any existing one; the frequency
  requires --target and excludes --target-date. --needs-whole-amount
  sets aside the full amount each period and --needs-whole-amount=false
  refills up to it; the API applies it to "plan your spending" targets
  only. On a credit card payment category the API supports only --target,
  so --target-frequency and --needs-whole-amount are refused there. A dry
  run previews the target's type, amount, and date from these rules and
  leaves the computed amounts as they were.`
