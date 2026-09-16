package cmd

import "github.com/spf13/cobra"

func outputFormatsTopic() *cobra.Command {
	return &cobra.Command{
		Use: "output-formats", Short: "Record shapes for --jsonl and --csv", Args: cobra.NoArgs,
		Long: `--jsonl writes one JSON object per line: one per record for list
commands, one line for get, status, and config. --csv, on list commands,
writes a header row of the same field names and one row per record with
the same values. Keys are lowercase snake_case. Amounts are JSON numbers
with two decimals written from the API's exact milliunit value ("450.00");
outflows are negative. Dates are YYYY-MM-DD, months YYYY-MM, timestamps
RFC 3339. Absent optional fields are omitted in JSONL and empty in CSV.
Enum values keep the API's spelling, such as "creditCard". Arrays such as
a split's subtransactions are omitted from CSV, which flattens them to
rows instead. Machine output never carries color.

Plan (plans list, plans get):
  id                 string
  name               string
  first_month        YYYY-MM
  last_month         YYYY-MM
  last_modified_on   timestamp
  currency           ISO 4217 code; omitted when the plan has no format
  date_format        such as "MM/DD/YYYY"; omitted when unavailable
  configured         true for the plan the configuration selects

Account (accounts list, accounts get, accounts create):
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
  dry_run                 true on the output of accounts create run with
                          --dry-run; omitted otherwise

Category (categories list, categories get, categories create,
categories update, months assign, move, cover, fund):
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
  dry_run             true on the output of a mutating command run with
                      --dry-run; omitted otherwise in JSONL and false in
                      CSV. A month write's dry run applies the change
                      arithmetically to assigned, available, and
                      underfunded; a category create or update's dry run
                      previews the target's type, amount, and date.
  In CSV the target object becomes target_type, target_amount, and so
  on, all empty when there is no target.

Category group (category-groups list, create, rename):
  id               string
  name             string
  hidden           boolean
  internal         boolean
  category_count   categories shown by categories list; every category
                   with --hidden; 0 on create
  dry_run          true on the output of a mutating command run with
                   --dry-run; omitted otherwise

Month (months list, months get):
  month             YYYY-MM
  note              string; omitted when empty
  income            amount
  assigned          amount
  activity          amount
  ready_to_assign   amount; negative when overassigned
  age_of_money      days; omitted until the plan has one

Transaction (transactions list, get, review, create, update, delete,
approve, categorize, clear, unclear, flag):
  id                          string; empty on a dry-run create
  parent_id                   CSV only: the parent's id on a split line,
                              empty on a transaction
  date                        YYYY-MM-DD
  account                     account name
  account_id                  string
  payee                       string; omitted when absent
  payee_id                    string; omitted when absent
  category                    category name, "Split" for a split;
                              omitted when uncategorized
  category_id                 string; omitted for splits and uncategorized
  memo                        string; omitted when empty
  amount                      amount
  cleared                     uncleared, cleared, or reconciled
  approved                    boolean
  flag_color                  red, orange, yellow, green, blue, purple;
                              omitted when unflagged
  flag_name                   the flag's custom name; omitted when unset
  transfer_account            account name; omitted unless a transfer
  transfer_account_id         string; omitted unless a transfer
  transfer_transaction_id     the other side; omitted unless a transfer
  matched_transaction_id      string; omitted unless matched
  import_id                   string; omitted unless imported
  import_payee_name           string; omitted unless imported
  import_payee_name_original  string; omitted unless imported
  debt_transaction_type       payment, refund, fee, interest, escrow,
                              balanceAdjustment, credit, charge; omitted
                              unless on a debt account
  subtransactions             array of lines; omitted unless a split
    id                          string
    payee                       string; omitted when the line inherits
    payee_id                    string; omitted when the line inherits
    category                    string; omitted when uncategorized
    category_id                 string; omitted when uncategorized
    memo                        string; omitted when empty
    amount                      amount
    transfer_account            account name; omitted unless a transfer
    transfer_account_id         string; omitted unless a transfer
    transfer_transaction_id     string; omitted unless a transfer
  needs                       transactions review only: approve,
                              categorize, or both
  dry_run                     true on the output of a mutating command
                              run with --dry-run; omitted otherwise in
                              JSONL and false in CSV
  In CSV each split line becomes its own row after the parent, with the
  line's id, payee, category, memo, amount, and transfer fields, the
  parent's id in parent_id, and the parent's date, account, cleared,
  approved, and flag.

Payee (payees list, payees get, payees create, payees rename):
  id                    string
  name                  string
  transfer_account      account name; omitted unless a transfer payee
  transfer_account_id   string; omitted unless a transfer payee
  dry_run               true on the output of a mutating command run
                        with --dry-run; omitted otherwise

Scheduled transaction (scheduled list, scheduled get):
  id                    string
  parent_id             CSV only, as for transactions
  next_date             YYYY-MM-DD
  first_date            YYYY-MM-DD
  frequency             never, daily, weekly, everyOtherWeek,
                        twiceAMonth, every4Weeks, monthly,
                        everyOtherMonth, every3Months, every4Months,
                        twiceAYear, yearly, everyOtherYear
  account               account name
  account_id            string
  payee                 string; omitted when absent
  payee_id              string; omitted when absent
  category              category name, "Split" for a split; omitted when
                        uncategorized
  category_id           string; omitted for splits and uncategorized
  memo                  string; omitted when empty
  amount                amount
  flag_color            as for transactions; omitted when unflagged
  flag_name             omitted when unset
  transfer_account      account name; omitted unless a transfer
  transfer_account_id   string; omitted unless a transfer
  subtransactions       array of lines with id, payee, payee_id,
                        category, category_id, memo, amount,
                        transfer_account, transfer_account_id; omitted
                        unless a split. CSV flattens as for transactions.

Money movement (money-movements list):
  id         string
  month      YYYY-MM; omitted when absent
  moved_at   timestamp; omitted when absent
  from       source category name, or "Ready to Assign"
  from_id    string; omitted for ready to assign
  to         target category name, or "Ready to Assign"
  to_id      string; omitted for ready to assign
  amount     amount
  note       string; omitted when empty
  group_id   the movement group; omitted when absent

Import (transactions import --jsonl):
  count             transactions imported, zero or more
  transaction_ids   array of their ids; empty when none

Status (plans status --jsonl):
  month                   YYYY-MM, the current month
  ready_to_assign         amount; negative when overassigned
  age_of_money            days; omitted until the plan has one
  overspent_count         categories with available below zero
  overspent_total         amount, the sum of those available amounts
  underfunded_count       targets still needing money this month
  underfunded_total       amount
  unapproved_count        integer
  uncategorized_count     integer
  import_error_accounts   array of open account names in direct import
                          error; empty when none

Funding report (reports funding):
  id                 string
  name               string
  group              category group name
  month              YYYY-MM
  target_type        TB, TBD, MF, NEED, or DEBT
  target_amount      amount
  assigned           amount
  activity           amount
  available          amount; negative when overspent
  underfunded        amount still needed this month; omitted when the
                     API reports none
  percent_complete   integer; omitted when absent
  months_to_assign   months left in the period; omitted when absent

Spending report (reports spending):
  id             category or payee id; omitted on the Uncategorized and
                 No payee rows
  name           category or payee name
  group          category group name; omitted with --by payee and on the
                 Uncategorized row
  outflows       amount, zero or negative
  inflows        amount, zero or positive
  net            amount, the sum of both
  transactions   transactions touching the row

Config (config --jsonl):
  allow_writes   boolean
  plan           string
  token          "<redacted>" when set, "" otherwise
  sources        object of field to source; only with --provenance

The api commands have no --jsonl or --csv: they print the API's response
body as returned, with milliunit amounts and the API's field names.`,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
}
