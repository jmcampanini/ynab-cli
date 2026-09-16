package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// Shared machinery for the month writes: the session with one month
// loaded, the ready-to-assign operand, the checks against available
// money, and the sequenced assigned-amount writes with their
// partial-failure report.

// readyToAssignOperand is the operand that names the plan's unassigned money
// where --from or --to expects a category.
const readyToAssignOperand = "ready-to-assign"

// monthSession is a write session with the categories endpoint and one
// month loaded, so every category's amounts for that month and the
// month's ready to assign are in hand before any write.
type monthSession struct {
	*writeSession
	apiMonth string
	month    ynab.Month
	rows     map[string]ynab.Category
}

// openMonth parses the month operand, applies the write gate, and reads
// the plan, the categories, and the month.
func (a *app) openMonth(cmd *cobra.Command, dryRun bool, monthText string) (*monthSession, error) {
	apiMonth, err := a.month(monthText)
	if err != nil {
		return nil, err
	}
	s, err := a.openWrite(cmd, dryRun)
	if err != nil {
		return nil, err
	}

	ctx := cmd.Context()
	if s.categories, err = s.client.Categories(ctx, s.plan.ID); err != nil {
		return nil, err
	}
	month, err := s.client.Month(ctx, s.plan.ID, apiMonth)
	if err != nil {
		return nil, err
	}
	return &monthSession{writeSession: s, apiMonth: apiMonth, month: month, rows: categoriesByID(month.Categories)}, nil
}

// shortMonth is the session's month as YYYY-MM.
func (ms *monthSession) shortMonth() string {
	return shortMonth(ms.apiMonth)
}

// monthCategory is a category with its group and its amounts for the
// session's month.
type monthCategory struct {
	category ynab.Category
	group    ynab.CategoryGroup
}

// qualifiedName is the "Group: Name" form, which names the category
// unambiguously in summaries and reversal commands.
func (m monthCategory) qualifiedName() string {
	return m.group.Name + ": " + m.category.Name
}

// category resolves an operand as 'categories get' does and returns the
// category with the month's amounts. Internal categories, which the API
// assigns to itself, are refused. Credit card payment categories are
// allowed, since assigning to them funds the card's payment.
func (ms *monthSession) category(query string) (monthCategory, error) {
	category, group, err := findCategory(query, ms.categories)
	if err != nil {
		return monthCategory{}, err
	}
	if category.Internal {
		return monthCategory{}, fmt.Errorf("category %q is internal; the API assigns to it itself", query)
	}
	row, ok := ms.rows[category.ID]
	if !ok {
		return monthCategory{}, fmt.Errorf("category %q is missing from month %s", category.Name, ms.shortMonth())
	}
	return monthCategory{category: row, group: group}, nil
}

// moneySource is where a move takes money from or sends it to: a
// category, or the month's ready to assign.
type moneySource struct {
	category      monthCategory
	readyToAssign bool
}

// source resolves a --from or --to operand.
func (ms *monthSession) source(query string) (moneySource, error) {
	if strings.EqualFold(query, readyToAssignOperand) {
		return moneySource{readyToAssign: true}, nil
	}
	category, err := ms.category(query)
	if err != nil {
		return moneySource{}, err
	}
	return moneySource{category: category}, nil
}

func (m moneySource) name() string {
	if m.readyToAssign {
		return "ready to assign"
	}
	return m.category.qualifiedName()
}

// same reports whether both operands name the same category.
func (m moneySource) same(other moneySource) bool {
	return !m.readyToAssign && !other.readyToAssign && m.category.category.ID == other.category.category.ID
}

// available is what the source can give: the category's available
// amount, or the month's ready to assign.
func (ms *monthSession) available(source moneySource) ynab.Amount {
	if source.readyToAssign {
		return ms.month.ReadyToAssign
	}
	return source.category.category.Available
}

// checkAvailable refuses taking amount from a source that has less,
// unless --force was given.
func (ms *monthSession) checkAvailable(source moneySource, amount ynab.Amount, force bool) error {
	available := ms.available(source)
	if force || available >= amount {
		return nil
	}
	currency := ms.plan.CurrencyFormat
	return fmt.Errorf("%s has %s available in %s, less than %s; pass --force to take it anyway", source.name(), available.Format(currency), ms.shortMonth(), amount.Format(currency))
}

// assignment is one assigned-amount write: the category's row before the
// write and the value to write.
type assignment struct {
	assigned ynab.Amount
	before   monthCategory
}

// delta is the change the write makes to the assigned amount.
func (w assignment) delta() ynab.Amount {
	return w.assigned - w.before.category.Assigned
}

// preview applies the write arithmetically, as a dry run shows it:
// available moves by the delta, and an underfunded amount shrinks by it
// down to zero.
func (w assignment) preview() ynab.Category {
	after := w.before.category
	delta := w.delta()
	after.Assigned = w.assigned
	after.Available += delta
	if after.TargetUnderfunded != nil {
		underfunded := max(0, *after.TargetUnderfunded-delta)
		after.TargetUnderfunded = &underfunded
	}
	return after
}

// reversal is the assign command that undoes the write, with the plan,
// the month, and --allow-writes spelled out so it runs as pasted from
// any shell. The category is named by ID, which needs no quoting.
func (w assignment) reversal(planID, month string, digits int) string {
	mode, amount := "--subtract", w.delta()
	if amount < 0 {
		mode, amount = "--add", -amount
	}
	return fmt.Sprintf("ynab months assign %s %s %s --plan %s --month %s --allow-writes", w.before.category.ID, amount.Decimal(digits), mode, planID, month)
}

// apply performs the writes in order and prints the resulting rows. A
// dry run previews every row without writing. When a later write fails,
// the rows already written are printed, and the error names each write
// applied with the command that reverses it, so nothing succeeds
// silently.
func (ms *monthSession) apply(cmd *cobra.Command, output outputFlags, writes []assignment, summary string) error {
	ctx := cmd.Context()
	var records []categoryRecord
	for i, write := range writes {
		after := write.preview()
		if !ms.dryRun {
			var err error
			after, err = ms.client.UpdateMonthCategory(ctx, ms.plan.ID, ms.apiMonth, write.before.category.ID, write.assigned)
			if err != nil {
				err = fmt.Errorf("write %d of %d, %s: %w", i+1, len(writes), write.before.qualifiedName(), err)
				if i == 0 {
					return err
				}
				return errors.Join(ms.partialFailure(err, writes[:i]), ms.printCategories(cmd, output, records, fmt.Sprintf("%s of %d applied before the failure", countNoun(i, "write", "writes"), len(writes))))
			}
		}
		records = append(records, newCategoryRecord(after, write.before.group, ms.apiMonth))
	}
	return ms.printCategories(cmd, output, records, summary)
}

// partialFailure extends a write error with the writes already applied
// and the commands that reverse them, in the order to run them.
func (ms *monthSession) partialFailure(err error, applied []assignment) error {
	reversals := make([]string, len(applied))
	for i, write := range applied {
		reversals[i] = write.reversal(ms.plan.ID, ms.shortMonth(), ms.decimalDigits())
	}
	return fmt.Errorf("%w; %s already applied, reverse with: %s", err, countNoun(len(applied), "write", "writes"), strings.Join(reversals, " && "))
}

// categoryWriteRecord is a category record printed by a mutating
// command, marked when it is a dry-run preview.
type categoryWriteRecord struct {
	categoryRecord
	DryRun bool `json:"dry_run,omitempty"`
}

func (s *writeSession) categoryWriteRecords(records []categoryRecord) []categoryWriteRecord {
	marked := make([]categoryWriteRecord, len(records))
	for i, record := range records {
		marked[i] = categoryWriteRecord{categoryRecord: record, DryRun: s.dryRun}
	}
	return marked
}

// printCategories writes the rows in the chosen format; human output is
// a flat table in the 'categories list' columns followed by the summary.
// Machine output carries no summary, so when there are no rows to
// print the summary goes to stderr as a diagnostic.
func (ms *monthSession) printCategories(cmd *cobra.Command, output outputFlags, records []categoryRecord, summary string) error {
	out := cmd.OutOrStdout()
	marked := ms.categoryWriteRecords(records)
	if (output.jsonl || output.csv) && len(records) == 0 {
		// Losing the diagnostic must not fail the command.
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), summary)
	}
	switch {
	case output.jsonl:
		return writeJSONL(out, marked)
	case output.csv:
		return writeCSV(out, marked)
	}

	colors := ms.app.palette(cmd)
	rows := make([][]cell, len(records))
	for i, record := range records {
		rows[i] = categoryRow(record, "", ms.plan.CurrencyFormat, colors)
	}
	if err := writeTable(out, categoryColumns, rows); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, summary)
	return err
}

// monthWriteHelp is the requests and checks paragraph the month writes
// share.
const monthWriteHelp = `Month writes:
  Every month write reads the plans endpoint, the categories endpoint,
  and the month, then sends one request per category changed, in the
  order the command states. Writes are not atomic: when a later one
  fails, the rows already written are printed, the exit status is 1, and
  the error names each write applied with the exact 'ynab months assign'
  command that reverses it, by category ID with the plan and month set.
  When there is nothing to write, the summary goes to stdout in human
  output and to stderr with --jsonl or --csv, which print no records. A
  dry run makes the reads, applies the arithmetic, and prints the rows as
  they would be: assigned and available move by the change, and an
  underfunded amount shrinks by it. CATEGORY is an ID, an exact name, or
  "Group: Name" as in 'categories get'; credit card payment categories
  are allowed, internal ones are not. ready-to-assign, matched ignoring
  case, names the month's unassigned money.`
