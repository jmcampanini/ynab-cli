package cmd

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// categoryListing is the category rows categories list and months get
// share: groups in plan order, each with the rows that survived the
// hidden filter, plus the count of rows the filter removed.
type categoryListing struct {
	groups    []groupListing
	hiddenOut int
}

// groupListing is one category group's shown rows.
type groupListing struct {
	hidden  bool
	name    string
	records []categoryRecord
}

// listCategories builds the rows for apiMonth from the plan's groups. A
// nil monthValues uses the current-month amounts the groups carry;
// otherwise each row takes its amounts from monthValues by ID. Hidden
// categories and every category of a hidden group are dropped unless
// includeHidden is set, and groups left without rows are omitted.
func listCategories(groups []ynab.CategoryGroup, monthValues map[string]ynab.Category, apiMonth string, includeHidden bool) (categoryListing, error) {
	var listing categoryListing
	for _, group := range groups {
		shown := groupListing{hidden: group.Hidden, name: group.Name}
		for _, category := range group.Categories {
			if (category.Hidden || group.Hidden) && !includeHidden {
				listing.hiddenOut++
				continue
			}
			if monthValues != nil {
				monthCategory, ok := monthValues[category.ID]
				if !ok {
					return categoryListing{}, fmt.Errorf("category %q is missing from month %s", category.Name, shortMonth(apiMonth))
				}
				category = monthCategory
			}
			shown.records = append(shown.records, newCategoryRecord(category, group, apiMonth))
		}
		if len(shown.records) > 0 {
			listing.groups = append(listing.groups, shown)
		}
	}
	return listing, nil
}

// records flattens the listing for machine output.
func (l categoryListing) records() []categoryRecord {
	var records []categoryRecord
	for _, group := range l.groups {
		records = append(records, group.records...)
	}
	return records
}

// summary is the count line under the table.
func (l categoryListing) summary() string {
	text := countNoun(len(l.records()), "category", "categories")
	if l.hiddenOut > 0 {
		text += fmt.Sprintf(", %d hidden not shown", l.hiddenOut)
	}
	return text
}

var categoryColumns = []column{
	{name: "NAME"},
	{name: "ASSIGNED", right: true}, {name: "ACTIVITY", right: true}, {name: "AVAILABLE", right: true},
	{name: "TARGET"}, {name: "UNDERFUNDED", right: true},
}

// rows renders each group as a row carrying the subtotals of its shown
// categories, followed by the categories indented beneath it. Negative
// available is red; hidden rows are faint with a "(hidden)" suffix.
func (l categoryListing) rows(currency *ynab.CurrencyFormat, colors palette) [][]cell {
	var rows [][]cell
	for _, group := range l.groups {
		var assigned, activity, available, underfunded ynab.Amount
		hasTarget := false
		for _, record := range group.records {
			assigned += record.Assigned
			activity += record.Activity
			available += record.Available
			if record.Target != nil && record.Target.Underfunded != nil {
				underfunded += *record.Target.Underfunded
				hasTarget = true
			}
		}
		groupRow := []cell{
			plain(hiddenSuffix(group.name, group.hidden)),
			plainAmount(assigned, currency), plainAmount(activity, currency), availableCell(available, currency, colors),
			plain(""), plain(""),
		}
		if hasTarget {
			groupRow[5] = plainAmount(underfunded, currency)
		}
		rows = append(rows, paintRow(groupRow, group.hidden, colors))

		for _, record := range group.records {
			row := []cell{
				plain("  " + hiddenSuffix(record.Name, record.Hidden)),
				plainAmount(record.Assigned, currency), plainAmount(record.Activity, currency), availableCell(record.Available, currency, colors),
				plain(record.Target.summary(currency)), plain(""),
			}
			if record.Target != nil && record.Target.Underfunded != nil {
				row[5] = plainAmount(*record.Target.Underfunded, currency)
			}
			rows = append(rows, paintRow(row, record.Hidden, colors))
		}
	}
	return rows
}

func hiddenSuffix(name string, hidden bool) string {
	if hidden {
		return name + " (hidden)"
	}
	return name
}

func plainAmount(amount ynab.Amount, currency *ynab.CurrencyFormat) cell {
	return plain(amount.Format(currency))
}

// availableCell paints an overspent amount red.
func availableCell(amount ynab.Amount, currency *ynab.CurrencyFormat, colors palette) cell {
	c := plainAmount(amount, currency)
	if amount < 0 {
		c.paint = colors.red
	}
	return c
}

// paintRow makes every cell of a hidden row faint.
func paintRow(row []cell, hidden bool, colors palette) []cell {
	if hidden {
		for i := range row {
			row[i].paint = colors.faint
		}
	}
	return row
}
