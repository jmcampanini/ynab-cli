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

Config (config --jsonl):
  allow_writes   boolean
  plan           string
  token          "<redacted>" when set, "" otherwise
  sources        object of field to source; only with --provenance`,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
}
