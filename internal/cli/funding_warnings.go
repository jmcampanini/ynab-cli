package cli

import (
	"fmt"

	"github.com/jmcampanini/ynab-cli/internal/categoryview"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func (a *app) warnFutureRefill(cmd *cobra.Command, records []categoryview.Record, currency *ynab.CurrencyFormat) {
	currentMonth := a.deps.now().UTC().Format("2006-01")
	for _, record := range records {
		target := record.Target
		if record.Month <= currentMonth || record.Internal || target == nil || target.Type != "NEED" ||
			target.NeedsWholeAmount == nil || *target.NeedsWholeAmount {
			continue
		}
		underfunded := "not reported"
		if target.Underfunded != nil {
			underfunded = target.Underfunded.Format(currency)
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
			"warning [future-refill-funding]: %s: %s (%s), %s: API underfunded %s. "+
				"YNAB's web app may still require funding because it ignores prior-period funding for future refill targets. "+
				"Check the web target panel before treating this category as funded.\n",
			record.Group, record.Name, record.ID, record.Month, underfunded)
	}
}
