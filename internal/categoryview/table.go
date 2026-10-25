package categoryview

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/dates"
	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// Listing is the category rows categories list and months get
// share: groups in plan order, each with the rows that survived the
// hidden filter, plus the count of rows the filter removed.
type Listing struct {
	groups    []groupListing
	hiddenOut int
}

// groupListing is one category group's shown rows.
type groupListing struct {
	hidden  bool
	name    string
	records []Record
}

// List builds the rows for apiMonth from the plan's groups. A
// nil monthValues uses the current-month amounts the groups carry;
// otherwise each row takes its amounts from monthValues by ID. Hidden
// categories and every category of a hidden group are dropped unless
// includeHidden is set, and groups left without rows are omitted.
func List(groups []ynab.CategoryGroup, monthValues map[string]ynab.Category, apiMonth string, includeHidden bool) (Listing, error) {
	var listing Listing
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
					return Listing{}, fmt.Errorf("category %q is missing from month %s", category.Name, dates.ShortMonth(apiMonth))
				}
				category = monthCategory
			}
			shown.records = append(shown.records, NewRecord(category, group, apiMonth))
		}
		if len(shown.records) > 0 {
			listing.groups = append(listing.groups, shown)
		}
	}
	return listing, nil
}

// Records flattens the listing for machine output.
func (l Listing) Records() []Record {
	var records []Record
	for _, group := range l.groups {
		records = append(records, group.records...)
	}
	return records
}

// Summary is the count line under the table.
func (l Listing) Summary() string {
	text := render.CountNoun(len(l.Records()), "category", "categories")
	if l.hiddenOut > 0 {
		text += fmt.Sprintf(", %d hidden not shown", l.hiddenOut)
	}
	return text
}

// Columns defines the shared category table layout.
var Columns = []render.Column{
	{Name: "NAME"},
	{Name: "ASSIGNED", Right: true}, {Name: "ACTIVITY", Right: true}, {Name: "AVAILABLE", Right: true},
	{Name: "TARGET"}, {Name: "UNDERFUNDED", Right: true},
}

// Rows renders each group as a row carrying the subtotals of its shown
// categories, followed by the categories indented beneath it. Negative
// available is red; hidden rows are faint with a "(hidden)" suffix.
func (l Listing) Rows(currency *ynab.CurrencyFormat, colors render.Palette) [][]render.Cell {
	var rows [][]render.Cell
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
		groupUnderfunded := render.Plain("")
		if hasTarget {
			groupUnderfunded = render.Amount(underfunded, currency)
		}
		rows = append(rows, render.PaintRow([]render.Cell{
			render.Plain(render.HiddenSuffix(group.name, group.hidden)),
			render.Amount(assigned, currency), render.Amount(activity, currency), render.Available(available, currency, colors),
			render.Plain(""), groupUnderfunded,
		}, group.hidden, colors))

		for _, record := range group.records {
			rows = append(rows, Row(record, "  ", currency, colors))
		}
	}
	return rows
}

// Row renders one category in Columns, its name
// prefixed by indent.
func Row(record Record, indent string, currency *ynab.CurrencyFormat, colors render.Palette) []render.Cell {
	underfunded := render.Plain("")
	if record.Target != nil && record.Target.Underfunded != nil {
		underfunded = render.Amount(*record.Target.Underfunded, currency)
	}
	return render.PaintRow([]render.Cell{
		render.Plain(indent + render.HiddenSuffix(record.Name, record.Hidden)),
		render.Amount(record.Assigned, currency), render.Amount(record.Activity, currency), render.Available(record.Available, currency, colors),
		render.Plain(record.Target.Summary(currency)), underfunded,
	}, record.Hidden, colors)
}
