// Package categoryview shapes and renders category amounts and targets across commands.
package categoryview

import (
	"fmt"
	"time"

	"github.com/jmcampanini/ynab-cli/internal/dates"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// Record is the --jsonl and --csv shape of a category in one
// month.
type Record struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Group     string      `json:"group"`
	GroupID   string      `json:"group_id"`
	Month     string      `json:"month"`
	Hidden    bool        `json:"hidden"`
	Internal  bool        `json:"internal"`
	Note      string      `json:"note,omitempty"`
	Assigned  ynab.Amount `json:"assigned"`
	Activity  ynab.Amount `json:"activity"`
	Available ynab.Amount `json:"available"`
	Target    *Target     `json:"target,omitempty"`
}

// Target is the target object of a category record, carrying the
// API's raw codes. CSV flattens it into target_* columns.
type Target struct {
	Type             string       `json:"type"`
	Amount           ynab.Amount  `json:"amount"`
	Date             string       `json:"date,omitempty"`
	Cadence          *int         `json:"cadence,omitempty"`
	CadenceFrequency *int         `json:"cadence_frequency,omitempty"`
	Day              *int         `json:"day,omitempty"`
	NeedsWholeAmount *bool        `json:"needs_whole_amount,omitempty"`
	CreatedMonth     string       `json:"created_month,omitempty"`
	PercentComplete  *int         `json:"percent_complete,omitempty"`
	MonthsToAssign   *int         `json:"months_to_assign,omitempty"`
	Underfunded      *ynab.Amount `json:"underfunded,omitempty"`
	OverallFunded    *ynab.Amount `json:"overall_funded,omitempty"`
	OverallLeft      *ynab.Amount `json:"overall_left,omitempty"`
	SnoozedAt        *time.Time   `json:"snoozed_at,omitempty"`
}

// NewRecord shapes a category whose amounts belong to apiMonth,
// the API's YYYY-MM-01 form. The group supplies the display name because
// the month endpoints do not always carry it.
func NewRecord(category ynab.Category, group ynab.CategoryGroup, apiMonth string) Record {
	record := Record{
		Activity:  category.Activity,
		Assigned:  category.Assigned,
		Available: category.Available,
		Group:     group.Name,
		GroupID:   group.ID,
		Hidden:    category.Hidden || group.Hidden,
		ID:        category.ID,
		Internal:  category.Internal,
		Month:     dates.ShortMonth(apiMonth),
		Name:      category.Name,
	}
	if category.Note != nil {
		record.Note = *category.Note
	}
	if category.TargetType != nil {
		record.Target = &Target{
			Amount:           category.TargetAmount,
			Cadence:          category.TargetCadence,
			CadenceFrequency: category.TargetCadenceFreq,
			Day:              category.TargetDay,
			MonthsToAssign:   category.TargetMonthsToAssign,
			NeedsWholeAmount: category.TargetNeedsWholeAmt,
			OverallFunded:    category.TargetOverallFunded,
			OverallLeft:      category.TargetOverallLeft,
			PercentComplete:  category.TargetPercentComplete,
			SnoozedAt:        category.TargetSnoozedAt,
			Type:             *category.TargetType,
			Underfunded:      category.TargetUnderfunded,
		}
		if category.TargetDate != nil {
			record.Target.Date = *category.TargetDate
		}
		if category.TargetCreationMonth != nil {
			record.Target.CreatedMonth = dates.ShortMonth(*category.TargetCreationMonth)
		}
	}
	return record
}

// Summary is the TARGET column: the type code and amount, or blank.
func (t *Target) Summary(currency *ynab.CurrencyFormat) string {
	if t == nil {
		return ""
	}
	return t.Type + " " + t.Amount.Format(currency)
}

// TargetTypeWords spells out the API's target type codes.
var TargetTypeWords = map[string]string{
	"DEBT": "debt payoff",
	"MF":   "monthly funding",
	"NEED": "plan your spending",
	"TB":   "target balance",
	"TBD":  "target balance by date",
}

// TypeWords decodes the type code, keeping the code beside the words.
func (t *Target) TypeWords() string {
	words, ok := TargetTypeWords[t.Type]
	if !ok {
		return t.Type
	}
	return words + " (" + t.Type + ")"
}

// CadenceWords decodes cadence and frequency as the API defines them:
// codes 0, 1, 2, and 13 repeat every frequency units of none, month, week,
// or year; codes 3 to 12 mean every 2 to 11 months; 14 means every 2 years.
func (t *Target) CadenceWords() string {
	if t.Cadence == nil {
		return ""
	}
	every := 1
	if t.CadenceFrequency != nil {
		every = *t.CadenceFrequency
	}
	switch code := *t.Cadence; {
	case code == 0:
		return "none"
	case code == 1:
		return everyWords("month", every)
	case code == 2:
		return everyWords("week", every)
	case code == 13:
		return everyWords("year", every)
	case code >= 3 && code <= 12:
		return everyWords("month", code-1)
	case code == 14:
		return everyWords("year", 2)
	}
	return fmt.Sprintf("cadence %d", *t.Cadence)
}

func everyWords(unit string, count int) string {
	if count <= 1 {
		return unit + "ly"
	}
	return fmt.Sprintf("every %d %ss", count, unit)
}

// DayWords decodes the due day: a weekday for weekly targets, otherwise a
// day of the month. The API sends null for the last day of the month on
// every other repeating cadence, and for targets that do not repeat.
func (t *Target) DayWords() string {
	weekly := t.Cadence != nil && *t.Cadence == 2
	if t.Day == nil {
		if t.Cadence == nil || *t.Cadence == 0 || weekly {
			return ""
		}
		return "last day of the month"
	}
	if weekly {
		return time.Weekday(*t.Day).String()
	}
	return fmt.Sprintf("day %d of the month", *t.Day)
}

// RolloverWords decodes needs_whole_amount, which only NEED targets carry.
func (t *Target) RolloverWords() string {
	switch {
	case t.NeedsWholeAmount == nil:
		return ""
	case *t.NeedsWholeAmount:
		return "set aside the full amount each period"
	}
	return "refill up to the amount"
}
