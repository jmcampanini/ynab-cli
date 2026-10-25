package cli

import (
	"strconv"

	"github.com/jmcampanini/ynab-cli/internal/categoryview"
	"github.com/jmcampanini/ynab-cli/internal/render"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

// categoryFields lists the record for human output, decoding the target
// to words and showing the target rows only when there is one.
func categoryFields(record categoryview.Record, currency *ynab.CurrencyFormat) [][2]string {
	fields := [][2]string{
		{"id", record.ID},
		{"name", record.Name},
		{"group", record.Group},
		{"month", record.Month},
		{"hidden", strconv.FormatBool(record.Hidden)},
		{"internal", strconv.FormatBool(record.Internal)},
		{"note", record.Note},
		{"assigned", record.Assigned.Format(currency)},
		{"activity", record.Activity.Format(currency)},
		{"available", record.Available.Format(currency)},
	}
	target := record.Target
	if target == nil {
		return append(fields, [2]string{"target", "none"})
	}
	return append(fields,
		[2]string{"target", target.TypeWords()},
		[2]string{"target amount", target.Amount.Format(currency)},
		[2]string{"target date", target.Date},
		[2]string{"target cadence", target.CadenceWords()},
		[2]string{"target day", target.DayWords()},
		[2]string{"target rollover", target.RolloverWords()},
		[2]string{"target created", target.CreatedMonth},
		[2]string{"target progress", optionalPercent(target.PercentComplete)},
		[2]string{"target months left", render.OptionalInt(target.MonthsToAssign)},
		[2]string{"target underfunded", optionalAmount(target.Underfunded, currency)},
		[2]string{"target funded", optionalAmount(target.OverallFunded, currency)},
		[2]string{"target left", optionalAmount(target.OverallLeft, currency)},
		[2]string{"target snoozed", render.OptionalTime(target.SnoozedAt)},
	)
}

func optionalAmount(amount *ynab.Amount, currency *ynab.CurrencyFormat) string {
	if amount == nil {
		return ""
	}
	return amount.Format(currency)
}

func optionalPercent(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value) + "%"
}
