package cmd

import "github.com/spf13/cobra"

func outputFormatsTopic() *cobra.Command {
	return &cobra.Command{
		Use: "output-formats", Short: "Record shapes for --jsonl and --csv", Args: cobra.NoArgs,
		Long: `--jsonl writes one JSON object per line: one per record for list
commands, one line for get and config. --csv, on list commands, writes a
header row of the same field names and one row per record with the same
values. Keys are lowercase snake_case. Amounts are JSON numbers with two
decimals written from the API's exact milliunit value ("450.00"); outflows
are negative. Dates are YYYY-MM-DD, months YYYY-MM, timestamps RFC 3339.
Absent optional fields are omitted in JSONL and empty in CSV. Enum values
keep the API's spelling, such as "creditCard". Machine output never
carries color.

Plan (plans list, plans get):
  id                 string
  name               string
  first_month        YYYY-MM
  last_month         YYYY-MM
  last_modified_on   timestamp
  currency           ISO 4217 code; omitted when the plan has no format
  date_format        such as "MM/DD/YYYY"; omitted when unavailable
  configured         true for the plan the configuration selects

Account (accounts list, accounts get):
  id                      string
  name                    string
  type                    checking, savings, cash, creditCard, lineOfCredit,
                          otherAsset, otherLiability, mortgage, autoLoan,
                          studentLoan, personalLoan, medicalDebt, otherDebt
  on_plan                 true for plan accounts, false for tracking
  closed                  boolean
  note                    string; omitted when empty
  balance                 amount
  cleared_balance         amount
  uncleared_balance       amount
  transfer_payee_id       string; omitted when absent
  direct_import_linked    boolean
  direct_import_in_error  boolean
  last_reconciled_at      timestamp; omitted when never reconciled

Category (categories list, categories get):
  id                  string
  name                string
  group               category group name
  group_id            string
  month               YYYY-MM the amounts belong to
  hidden              true for a hidden category or one in a hidden group
  internal            true for the API's own categories
  note                string; omitted when empty
  assigned            amount
  activity            amount
  available           amount; negative when overspent
  target              object; omitted when the category has no target
    type                TB, TBD, MF, NEED, or DEBT
    amount              amount
    date                YYYY-MM-DD; omitted unless the type has one
    cadence             0 none, 1 monthly, 2 weekly, 3-12 every 2-11
                        months, 13 yearly, 14 every 2 years; omitted
                        when absent
    cadence_frequency   repeats every N units for cadence 0, 1, 2, 13;
                        omitted when absent
    day                 weekday (0 Sunday) for weekly, else day of month;
                        omitted when absent
    needs_whole_amount  boolean, NEED only; omitted otherwise
    created_month       YYYY-MM; omitted when absent
    percent_complete    integer; omitted when absent
    months_to_assign    months left in the period; omitted when absent
    underfunded         amount still needed this month; omitted when absent
    overall_funded      amount; omitted when absent
    overall_left        amount; omitted when absent
    snoozed_at          timestamp; omitted unless snoozed
  In CSV the target object becomes target_type, target_amount, and so
  on, all empty when there is no target.

Category group (category-groups list):
  id               string
  name             string
  hidden           boolean
  internal         boolean
  category_count   categories shown by categories list; every category
                   with --hidden

Month (months list, months get):
  month             YYYY-MM
  note              string; omitted when empty
  income            amount
  assigned          amount
  activity          amount
  ready_to_assign   amount; negative when overassigned
  age_of_money      days; omitted until the plan has one

Config (config --jsonl):
  allow_writes   boolean
  plan           string
  token          "<redacted>" when set, "" otherwise
  sources        object of field to source; only with --provenance`,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
}
