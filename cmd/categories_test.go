package cmd

import (
	"reflect"
	"strings"
	"testing"
)

func TestCategoriesListGroupsWithSubtotals(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "categories", "list")
	if err != nil {
		t.Fatal(err)
	}

	want := `NAME                        ASSIGNED    ACTIVITY  AVAILABLE  TARGET        UNDERFUNDED
Bills                      $1,580.00  -$1,579.99      $0.01                      $0.00
  Internet                    $80.00     -$79.99      $0.01  NEED $80.00         $0.00
  Rent                     $1,500.00  -$1,500.00      $0.00  MF $1,500.00        $0.00
Fun                          $200.00    -$225.00    -$25.00
  Dining Out                 $200.00    -$225.00    -$25.00
Credit Card Payments          $97.81      $97.81     $97.81
  Visa                        $97.81      $97.81     $97.81
Internal Master Category       $0.00       $0.00      $0.00
  Inflow: Ready to Assign      $0.00       $0.00      $0.00
5 categories, 2 hidden not shown
`
	if out != want {
		t.Errorf("categories list =\n%s\nwant\n%s", out, want)
	}
	if want := []string{"/plans", "/plans/p1/categories"}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("requests = %v, want %v", h.requests, want)
	}
}

func TestCategoriesListMonthAndHidden(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "categories", "list", "--month", "2026-08", "--hidden")
	if err != nil {
		t.Fatal(err)
	}

	want := `NAME                        ASSIGNED    ACTIVITY  AVAILABLE  TARGET        UNDERFUNDED
Bills                      $1,580.00  -$1,580.00      $0.00                      $0.00
  Internet                    $80.00     -$80.00      $0.00  NEED $80.00         $0.00
  Rent                     $1,500.00  -$1,500.00      $0.00  MF $1,500.00        $0.00
Fun                          $200.00    -$180.00     $25.00
  Dining Out                 $200.00    -$180.00     $20.00
  Old Hobby (hidden)           $0.00       $0.00      $5.00
Wishes (hidden)                $0.00       $0.00    $100.00                    $100.00
  Internet (hidden)            $0.00       $0.00    $100.00  TBD $500.00       $100.00
Credit Card Payments          $50.00      $50.00     $50.00
  Visa                        $50.00      $50.00     $50.00
Internal Master Category       $0.00       $0.00      $0.00
  Inflow: Ready to Assign      $0.00       $0.00      $0.00
7 categories
`
	if out != want {
		t.Errorf("categories list --month 2026-08 --hidden =\n%s\nwant\n%s", out, want)
	}
	if want := []string{"/plans", "/plans/p1/categories", "/plans/p1/months/2026-08-01"}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("requests = %v, want %v", h.requests, want)
	}
}

func TestCategoriesListMachineFormats(t *testing.T) {
	h := newHarness(t)
	h.terminal = true

	jsonl, err := h.execute(t, "categories", "list", "--jsonl", "--color", "always")
	if err != nil {
		t.Fatal(err)
	}
	csv, err := h.execute(t, "categories", "list", "--csv", "--color", "always")
	if err != nil {
		t.Fatal(err)
	}
	august, err := h.execute(t, "categories", "list", "--month", "2026-08-01", "--hidden", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}

	wantJSONL := `{"id":"c1","name":"Internet","group":"Bills","group_id":"g1","month":"2026-09","hidden":false,"internal":false,"note":"Fiber","assigned":80.00,"activity":-79.99,"available":0.01,"target":{"type":"NEED","amount":80.00,"cadence":1,"cadence_frequency":1,"needs_whole_amount":true,"created_month":"2025-08","percent_complete":100,"months_to_assign":1,"underfunded":0.00,"overall_funded":80.00,"overall_left":0.00}}
{"id":"c2","name":"Rent","group":"Bills","group_id":"g1","month":"2026-09","hidden":false,"internal":false,"assigned":1500.00,"activity":-1500.00,"available":0.00,"target":{"type":"MF","amount":1500.00,"cadence":1,"cadence_frequency":1,"created_month":"2024-01","percent_complete":100,"months_to_assign":1,"underfunded":0.00,"overall_funded":1500.00,"overall_left":0.00}}
{"id":"c3","name":"Dining Out","group":"Fun","group_id":"g2","month":"2026-09","hidden":false,"internal":false,"assigned":200.00,"activity":-225.00,"available":-25.00}
{"id":"c6","name":"Visa","group":"Credit Card Payments","group_id":"g4","month":"2026-09","hidden":false,"internal":false,"assigned":97.81,"activity":97.81,"available":97.81}
{"id":"c7","name":"Inflow: Ready to Assign","group":"Internal Master Category","group_id":"g5","month":"2026-09","hidden":false,"internal":true,"assigned":0.00,"activity":0.00,"available":0.00}
`
	if jsonl != wantJSONL {
		t.Errorf("jsonl =\n%s\nwant\n%s", jsonl, wantJSONL)
	}
	wantCSV := `id,name,group,group_id,month,hidden,internal,note,assigned,activity,available,target_type,target_amount,target_date,target_cadence,target_cadence_frequency,target_day,target_needs_whole_amount,target_created_month,target_percent_complete,target_months_to_assign,target_underfunded,target_overall_funded,target_overall_left,target_snoozed_at
c1,Internet,Bills,g1,2026-09,false,false,Fiber,80.00,-79.99,0.01,NEED,80.00,,1,1,,true,2025-08,100,1,0.00,80.00,0.00,
c2,Rent,Bills,g1,2026-09,false,false,,1500.00,-1500.00,0.00,MF,1500.00,,1,1,,,2024-01,100,1,0.00,1500.00,0.00,
c3,Dining Out,Fun,g2,2026-09,false,false,,200.00,-225.00,-25.00,,,,,,,,,,,,,,
c6,Visa,Credit Card Payments,g4,2026-09,false,false,,97.81,97.81,97.81,,,,,,,,,,,,,,
c7,Inflow: Ready to Assign,Internal Master Category,g5,2026-09,false,true,,0.00,0.00,0.00,,,,,,,,,,,,,,
`
	if csv != wantCSV {
		t.Errorf("csv =\n%s\nwant\n%s", csv, wantCSV)
	}
	wantAugustRow := `{"id":"c5","name":"Internet","group":"Wishes","group_id":"g3","month":"2026-08","hidden":true,"internal":false,"assigned":0.00,"activity":0.00,"available":100.00,"target":{"type":"TBD","amount":500.00,"date":"2027-01-01","created_month":"2026-05","percent_complete":20,"months_to_assign":5,"underfunded":100.00,"overall_funded":100.00,"overall_left":400.00}}`
	if !strings.Contains(august, wantAugustRow+"\n") || strings.Count(august, "\n") != 7 {
		t.Errorf("august jsonl =\n%s\nwant 7 rows including\n%s", august, wantAugustRow)
	}
}

func TestCategoriesListColor(t *testing.T) {
	h := newHarness(t)
	h.terminal = true

	out, err := h.execute(t, "categories", "list", "--hidden")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "\x1b[31m-$25.00\x1b[0m") {
		t.Errorf("overspent available is not red:\n%q", out)
	}
	if strings.Contains(out, "\x1b[31m-$225.00") || strings.Contains(out, "\x1b[31m-$1,579.99") {
		t.Errorf("negative activity is red:\n%q", out)
	}
	if !strings.Contains(out, "\x1b[2m  Old Hobby (hidden)\x1b[0m") || !strings.Contains(out, "\x1b[2mWishes (hidden)\x1b[0m") {
		t.Errorf("hidden rows are not faint:\n%q", out)
	}
}

func TestCategoriesGetResolvesNamesAndDecodesTarget(t *testing.T) {
	h := newHarness(t)

	byQualifiedName, err := h.execute(t, "categories", "get", "bills: internet")
	if err != nil {
		t.Fatal(err)
	}
	byUniqueName, err := h.execute(t, "categories", "get", "Dining Out")
	if err != nil {
		t.Fatal(err)
	}
	colonInName, err := h.execute(t, "categories", "get", "inflow: ready to assign", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	hidden, err := h.execute(t, "categories", "get", "old hobby", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}

	want := `id                  c1
name                Internet
group               Bills
month               2026-09
hidden              false
internal            false
note                Fiber
assigned            $80.00
activity            -$79.99
available           $0.01
target              plan your spending (NEED)
target amount       $80.00
target date
target cadence      monthly
target day          last day of the month
target rollover     set aside the full amount each period
target created      2025-08
target progress     100%
target months left  1
target underfunded  $0.00
target funded       $80.00
target left         $0.00
target snoozed
`
	if byQualifiedName != want {
		t.Errorf("categories get =\n%s\nwant\n%s", byQualifiedName, want)
	}
	if !strings.HasPrefix(byUniqueName, "id         c3\n") || !strings.HasSuffix(byUniqueName, "available  -$25.00\ntarget     none\n") {
		t.Errorf("categories get without a target =\n%s", byUniqueName)
	}
	if !strings.HasPrefix(colonInName, `{"id":"c7",`) {
		t.Errorf("a name containing a colon did not resolve as a whole: %s", colonInName)
	}
	if !strings.HasPrefix(hidden, `{"id":"c4",`) || !strings.Contains(hidden, `"hidden":true`) {
		t.Errorf("a hidden category did not resolve: %s", hidden)
	}
}

func TestCategoriesGetMonthUsesTheMonthCategoryEndpoint(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "categories", "get", "c3", "--month", "2026-08", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}

	want := `{"id":"c3","name":"Dining Out","group":"Fun","group_id":"g2","month":"2026-08","hidden":false,"internal":false,"assigned":200.00,"activity":-180.00,"available":20.00}
`
	if out != want {
		t.Errorf("categories get --month =\n%s\nwant\n%s", out, want)
	}
	if want := []string{"/plans", "/plans/p1/categories", "/plans/p1/months/2026-08-01/categories/c3"}; !reflect.DeepEqual(h.requests, want) {
		t.Errorf("requests = %v, want %v", h.requests, want)
	}
}

func TestCategoriesGetAmbiguousAndMissingNames(t *testing.T) {
	h := newHarness(t)

	_, ambiguous := h.execute(t, "categories", "get", "Internet")
	_, missing := h.execute(t, "categories", "get", "Internets")

	wantAmbiguous := `category "Internet" is ambiguous; use an ID or the full name: "Bills: Internet" (c1), "Wishes: Internet" (c5)`
	if ambiguous == nil || ambiguous.Error() != wantAmbiguous || ExitCode(ambiguous) != ExitFailure {
		t.Errorf("shared name error = %v (exit %d), want %q", ambiguous, ExitCode(ambiguous), wantAmbiguous)
	}
	if missing == nil || !strings.HasPrefix(missing.Error(), `category "Internets" not found; closest names: "Bills: Internet", "Wishes: Internet", `) {
		t.Errorf("miss error = %v", missing)
	}
}

func TestTargetWords(t *testing.T) {
	intPtr := func(v int) *int { return &v }
	cases := []struct {
		name   string
		target targetRecord
		want   [2]string
	}{
		{"monthly", targetRecord{Cadence: intPtr(1), CadenceFrequency: intPtr(1)}, [2]string{"monthly", "last day of the month"}},
		{"every other month", targetRecord{Cadence: intPtr(1), CadenceFrequency: intPtr(2), Day: intPtr(15)}, [2]string{"every 2 months", "day 15 of the month"}},
		{"weekly on friday", targetRecord{Cadence: intPtr(2), CadenceFrequency: intPtr(1), Day: intPtr(5)}, [2]string{"weekly", "Friday"}},
		{"weekly without a day", targetRecord{Cadence: intPtr(2)}, [2]string{"weekly", ""}},
		{"every 3 months by code", targetRecord{Cadence: intPtr(4)}, [2]string{"every 3 months", "last day of the month"}},
		{"yearly", targetRecord{Cadence: intPtr(13), CadenceFrequency: intPtr(1)}, [2]string{"yearly", "last day of the month"}},
		{"every 2 years", targetRecord{Cadence: intPtr(14)}, [2]string{"every 2 years", "last day of the month"}},
		{"none", targetRecord{Cadence: intPtr(0)}, [2]string{"none", ""}},
		{"absent", targetRecord{}, [2]string{"", ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := [2]string{tc.target.cadenceWords(), tc.target.dayWords()}

			if got != tc.want {
				t.Errorf("cadenceWords, dayWords = %q, want %q", got, tc.want)
			}
		})
	}
}
