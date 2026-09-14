package cmd

import (
	"fmt"
	"strings"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

// statusRecord is the --jsonl shape of the plan's one-screen summary for
// the current month.
type statusRecord struct {
	Month               string      `json:"month"`
	ReadyToAssign       ynab.Amount `json:"ready_to_assign"`
	AgeOfMoney          *int        `json:"age_of_money,omitempty"`
	OverspentCount      int         `json:"overspent_count"`
	OverspentTotal      ynab.Amount `json:"overspent_total"`
	UnderfundedCount    int         `json:"underfunded_count"`
	UnderfundedTotal    ynab.Amount `json:"underfunded_total"`
	UnapprovedCount     int         `json:"unapproved_count"`
	UncategorizedCount  int         `json:"uncategorized_count"`
	ImportErrorAccounts []string    `json:"import_error_accounts"`
}

// planStatus is the summary plus the names the human view lists.
type planStatus struct {
	overspent   []categoryRecord
	record      statusRecord
	underfunded []categoryRecord
}

// newPlanStatus derives the summary. Overspent is available below zero
// and underfunded is a target still needing money this month, both over
// the categories 'categories list' shows, which leaves out hidden ones
// and the internal inflow category. Closed accounts in import error are
// left out.
func newPlanStatus(month ynab.Month, groups []ynab.CategoryGroup, accounts []ynab.Account, unapproved, uncategorized []ynab.Transaction) (planStatus, error) {
	listing, err := listCategories(groups, categoriesByID(month.Categories), month.Month, false)
	if err != nil {
		return planStatus{}, err
	}
	status := planStatus{record: statusRecord{
		AgeOfMoney:          month.AgeOfMoney,
		ImportErrorAccounts: []string{},
		Month:               shortMonth(month.Month),
		ReadyToAssign:       month.ReadyToAssign,
		UnapprovedCount:     len(unapproved),
		UncategorizedCount:  len(uncategorized),
	}}
	for _, record := range listing.records() {
		if record.Internal {
			continue
		}
		if record.Available < 0 {
			status.overspent = append(status.overspent, record)
			status.record.OverspentCount++
			status.record.OverspentTotal += record.Available
		}
		if record.Target != nil && record.Target.Underfunded != nil && *record.Target.Underfunded > 0 {
			status.underfunded = append(status.underfunded, record)
			status.record.UnderfundedCount++
			status.record.UnderfundedTotal += *record.Target.Underfunded
		}
	}
	for _, account := range accounts {
		if account.DirectImportInError && !account.Closed {
			status.record.ImportErrorAccounts = append(status.record.ImportErrorAccounts, account.Name)
		}
	}
	return status, nil
}

// fields renders the summary for human output, naming the categories and
// accounts behind each count.
func (s planStatus) fields(currency *ynab.CurrencyFormat) [][2]string {
	record := s.record
	overspent := "none"
	if record.OverspentCount > 0 {
		var names []string
		for _, category := range s.overspent {
			names = append(names, category.Name+" "+category.Available.Format(currency))
		}
		overspent = fmt.Sprintf("%s, %s: %s", countNoun(record.OverspentCount, "category", "categories"), record.OverspentTotal.Format(currency), strings.Join(names, ", "))
	}
	underfunded := "none"
	if record.UnderfundedCount > 0 {
		var names []string
		for _, category := range s.underfunded {
			names = append(names, category.Name+" "+category.Target.Underfunded.Format(currency))
		}
		underfunded = fmt.Sprintf("%s, %s: %s", countNoun(record.UnderfundedCount, "target", "targets"), record.UnderfundedTotal.Format(currency), strings.Join(names, ", "))
	}
	importErrors := "none"
	if len(record.ImportErrorAccounts) > 0 {
		importErrors = strings.Join(record.ImportErrorAccounts, ", ")
	}
	return [][2]string{
		{"month", record.Month},
		{"ready to assign", record.ReadyToAssign.Format(currency)},
		{"age of money", optionalInt(record.AgeOfMoney)},
		{"overspent", overspent},
		{"underfunded", underfunded},
		{"unapproved", transactionCount(record.UnapprovedCount)},
		{"uncategorized", transactionCount(record.UncategorizedCount)},
		{"import errors", importErrors},
	}
}

func newPlansStatus(a *app) *cobra.Command {
	var output outputFlags
	command := &cobra.Command{
		Use: "status", Short: "Show what needs attention in the current month",
		Long: `Show the configured plan's current month on one screen: ready to
assign, age of money, the overspent categories with their count and
total, the targets still needing money with their count and total, the
counts of unapproved and uncategorized transactions, and the open
accounts whose direct import is in error. Overspent and underfunded are
taken over the categories 'categories list' shows, so hidden categories
are left out; Credit Card Payments categories count. Six requests: the
plans endpoint, the current month, the categories endpoint, the accounts
endpoint, then the unapproved and uncategorized listings. --jsonl prints
one object with the counts and totals and the import-error account
names. Exit 0 whether or not anything needs attention.

` + planHelp + "\n\n" + configHelp + "\n\n" + outputHelp,
		Example: `  ynab plans status
  ynab plans status --jsonl | jq .overspent_count`,
		Args: cobra.NoArgs,
		RunE: run(func(cmd *cobra.Command, _ []string) error {
			apiMonth, err := a.month("current")
			if err != nil {
				return err
			}
			loaded, client, err := a.connect(cmd)
			if err != nil {
				return err
			}
			plan, err := selectedPlan(cmd.Context(), client, loaded.Config)
			if err != nil {
				return err
			}
			month, err := client.Month(cmd.Context(), plan.ID, apiMonth)
			if err != nil {
				return err
			}
			groups, err := client.Categories(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}
			accounts, err := client.Accounts(cmd.Context(), plan.ID)
			if err != nil {
				return err
			}
			unapproved, err := client.Transactions(cmd.Context(), plan.ID, ynab.TransactionFilter{Type: ynab.TypeUnapproved})
			if err != nil {
				return err
			}
			uncategorized, err := client.Transactions(cmd.Context(), plan.ID, ynab.TransactionFilter{Type: ynab.TypeUncategorized})
			if err != nil {
				return err
			}

			status, err := newPlanStatus(month, groups, accounts, unapproved, uncategorized)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if output.jsonl {
				return writeJSONL(out, []statusRecord{status.record})
			}
			return writeFields(out, status.fields(plan.CurrencyFormat))
		}),
	}
	output.bind(command, false)
	return command
}
