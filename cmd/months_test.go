package cmd

import (
	"reflect"
	"strings"
	"testing"
)

func TestMonthsList(t *testing.T) {
	h := newHarness(t)
	h.terminal = true

	table, err := h.execute(t, "months", "list", "--color", "never")
	if err != nil {
		t.Fatal(err)
	}
	colored, err := h.execute(t, "months", "list")
	if err != nil {
		t.Fatal(err)
	}
	jsonl, err := h.execute(t, "months", "list", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	csv, err := h.execute(t, "months", "list", "--csv")
	if err != nil {
		t.Fatal(err)
	}

	wantTable := `MONTH       INCOME   ASSIGNED    ACTIVITY  READY TO ASSIGN  AGE OF MONEY
2026-08  $3,000.00  $1,780.00  -$1,710.00        $1,220.00            30
2026-09      $0.00  $1,780.00  -$1,707.18         -$500.00
2 months
`
	if table != wantTable {
		t.Errorf("months list =\n%s\nwant\n%s", table, wantTable)
	}
	if !strings.Contains(colored, "\x1b[31m-$500.00\x1b[0m") || strings.Contains(colored, "\x1b[31m-$1,7") {
		t.Errorf("only negative ready to assign should be red:\n%q", colored)
	}
	wantJSONL := `{"month":"2026-08","note":"August","income":3000.00,"assigned":1780.00,"activity":-1710.00,"ready_to_assign":1220.00,"age_of_money":30}
{"month":"2026-09","income":0.00,"assigned":1780.00,"activity":-1707.18,"ready_to_assign":-500.00}
`
	if jsonl != wantJSONL {
		t.Errorf("jsonl =\n%s\nwant\n%s", jsonl, wantJSONL)
	}
	wantCSV := `month,note,income,assigned,activity,ready_to_assign,age_of_money
2026-08,August,3000.00,1780.00,-1710.00,1220.00,30
2026-09,,0.00,1780.00,-1707.18,-500.00,
`
	if csv != wantCSV {
		t.Errorf("csv =\n%s\nwant\n%s", csv, wantCSV)
	}
}

func TestMonthsGetShowsTotalsThenCategoryRows(t *testing.T) {
	h := newHarness(t)

	august, err := h.execute(t, "months", "get", "2026-08")
	if err != nil {
		t.Fatal(err)
	}
	current, err := h.execute(t, "months", "get", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}

	wantAugust := `month            2026-08
note             August
income           $3,000.00
assigned         $1,780.00
activity         -$1,710.00
ready to assign  $1,220.00
age of money     30

NAME                        ASSIGNED    ACTIVITY  AVAILABLE  TARGET        UNDERFUNDED
Bills                      $1,580.00  -$1,580.00      $0.00                      $0.00
  Internet                    $80.00     -$80.00      $0.00  NEED $80.00         $0.00
  Rent                     $1,500.00  -$1,500.00      $0.00  MF $1,500.00        $0.00
Fun                          $200.00    -$180.00     $20.00
  Dining Out                 $200.00    -$180.00     $20.00
Credit Card Payments          $50.00      $50.00     $50.00
  Visa                        $50.00      $50.00     $50.00
Internal Master Category       $0.00       $0.00      $0.00
  Inflow: Ready to Assign      $0.00       $0.00      $0.00
5 categories, 2 hidden not shown
`
	if august != wantAugust {
		t.Errorf("months get 2026-08 =\n%s\nwant\n%s", august, wantAugust)
	}
	wantCurrent := `{"month":"2026-09","income":0.00,"assigned":1780.00,"activity":-1707.18,"ready_to_assign":-500.00}
`
	if current != wantCurrent {
		t.Errorf("months get --jsonl =\n%s\nwant\n%s", current, wantCurrent)
	}
	wantRequests := []string{
		"/plans", "/plans/p1/categories", "/plans/p1/months/2026-08-01",
		"/plans", "/plans/p1/categories", "/plans/p1/months/2026-09-01",
	}
	if !reflect.DeepEqual(h.requests, wantRequests) {
		t.Errorf("requests = %v, want %v", h.requests, wantRequests)
	}
}

func TestInvalidMonthIsAUsageErrorBeforeAnyRequest(t *testing.T) {
	h := newHarness(t)
	cases := [][]string{
		{"months", "get", "2026-08-15"}, {"months", "get", "august"},
		{"categories", "list", "--month", "2026-8"}, {"categories", "get", "Rent", "--month", "today"},
	}

	for _, args := range cases {
		out, err := h.execute(t, args...)
		if err == nil || out != "" || ExitCode(err) != ExitUsage || !strings.Contains(err.Error(), "YYYY-MM") {
			t.Errorf("execute(%v) = %q, %v (exit %d), want a usage error naming the forms", args, out, err, ExitCode(err))
		}
	}
	if len(h.requests) != 0 {
		t.Errorf("invalid months made requests: %v", h.requests)
	}
}
