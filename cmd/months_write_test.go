package cmd

import (
	"reflect"
	"strings"
	"testing"
)

func TestMonthsAssignWritesTheAssignedAmount(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		wantWrite string
		wantErr   string
	}{
		{"set", []string{"Dining Out", "450"}, `PATCH /plans/p1/months/2026-09-01/categories/c3 {"category":{"budgeted":450000}}`, ""},
		{"set negative", []string{"Dining Out", "--", "-5"}, `PATCH /plans/p1/months/2026-09-01/categories/c3 {"category":{"budgeted":-5000}}`, ""},
		{"add in another month", []string{"Rent", "20", "--add", "--month", "2026-08"}, `PATCH /plans/p1/months/2026-08-01/categories/c2 {"category":{"budgeted":1520000}}`, ""},
		{"subtract", []string{"Bills: Rent", "20.50", "--subtract"}, `PATCH /plans/p1/months/2026-09-01/categories/c2 {"category":{"budgeted":1479500}}`, ""},
		{"credit card payment category", []string{"Credit Card Payments: Visa", "10"}, `PATCH /plans/p1/months/2026-09-01/categories/c6 {"category":{"budgeted":10000}}`, ""},
		{"add a negative amount", []string{"Rent", "--add", "--", "-5"}, "", "positive amount"},
		{"internal category", []string{"Inflow: Ready to Assign", "5"}, "", "internal"},
		{"unknown category", []string{"Grocery", "5"}, "", `category "Grocery" not found`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)

			_, err := h.execute(t, append([]string{"months", "assign", "--allow-writes"}, tc.args...)...)

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) || len(h.writes) != 0 {
					t.Fatalf("assign %v = %v, writes %v; want an error containing %q and no write", tc.args, err, h.writes, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(h.writes, []string{tc.wantWrite}) {
				t.Errorf("writes = %v, want [%s]", h.writes, tc.wantWrite)
			}
		})
	}
}

func TestMonthsAssignPrintsTheStoredRowAfterThreeReads(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "months", "assign", "Dining Out", "450", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}

	want := `NAME        ASSIGNED  ACTIVITY  AVAILABLE  TARGET  UNDERFUNDED
Dining Out   $450.00  -$225.00    $225.00
assigned $450.00 to Fun: Dining Out for 2026-09
`
	if out != want {
		t.Errorf("assign =\n%s\nwant\n%s", out, want)
	}
	wantRequests := []string{"/plans", "/plans/p1/categories", "/plans/p1/months/2026-09-01", "/plans/p1/months/2026-09-01/categories/c3"}
	if !reflect.DeepEqual(h.requests, wantRequests) {
		t.Errorf("requests = %v, want %v", h.requests, wantRequests)
	}

	jsonl, err := h.execute(t, "months", "assign", "Rent", "20", "--add", "--dry-run", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	wantJSONL := `{"id":"c2","name":"Rent","group":"Bills","group_id":"g1","month":"2026-09","hidden":false,"internal":false,"assigned":1520.00,"activity":-1500.00,"available":20.00,"target":{"type":"MF","amount":1500.00,"cadence":1,"cadence_frequency":1,"created_month":"2024-01","percent_complete":100,"months_to_assign":1,"underfunded":0.00,"overall_funded":1500.00,"overall_left":0.00},"dry_run":true}` + "\n"
	if jsonl != wantJSONL {
		t.Errorf("assign --add --dry-run --jsonl =\n%s\nwant\n%s", jsonl, wantJSONL)
	}
	if len(h.writes) != 1 {
		t.Errorf("writes after the dry run = %v, want only the first run's", h.writes)
	}
}

func TestMonthsMoveWritesSourceThenTarget(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantWrites []string
		wantErr    string
	}{
		{
			"between categories", []string{"10", "--from", "Visa", "--to", "Rent"},
			[]string{`PATCH /plans/p1/months/2026-09-01/categories/c6 {"category":{"budgeted":87810}}`, `PATCH /plans/p1/months/2026-09-01/categories/c2 {"category":{"budgeted":1510000}}`}, "",
		},
		{
			"to ready to assign", []string{"10", "--from", "Visa", "--to", "Ready-To-Assign"},
			[]string{`PATCH /plans/p1/months/2026-09-01/categories/c6 {"category":{"budgeted":87810}}`}, "",
		},
		{
			"from ready to assign forced", []string{"10", "--from", "ready-to-assign", "--to", "Rent", "--force"},
			[]string{`PATCH /plans/p1/months/2026-09-01/categories/c2 {"category":{"budgeted":1510000}}`}, "",
		},
		{
			"past available forced", []string{"50", "--from", "Dining Out", "--to", "Rent", "--force", "--month", "2026-08"},
			[]string{`PATCH /plans/p1/months/2026-08-01/categories/c3 {"category":{"budgeted":150000}}`, `PATCH /plans/p1/months/2026-08-01/categories/c2 {"category":{"budgeted":1550000}}`}, "",
		},
		{"from ready to assign below the amount", []string{"10", "--from", "ready-to-assign", "--to", "Rent"}, nil, "ready to assign has -$500.00 available in 2026-09, less than $10.00; pass --force"},
		{"past available", []string{"50", "--from", "Dining Out", "--to", "Rent"}, nil, "Fun: Dining Out has -$25.00 available in 2026-09, less than $50.00; pass --force"},
		{"both ready to assign", []string{"10", "--from", "ready-to-assign", "--to", "Ready-To-Assign"}, nil, "both ready-to-assign"},
		{"same category", []string{"10", "--from", "Rent", "--to", "Bills: Rent"}, nil, "both name Bills: Rent"},
		{"zero amount", []string{"0", "--from", "Visa", "--to", "Rent"}, nil, "AMOUNT must be positive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)

			_, err := h.execute(t, append([]string{"months", "move", "--allow-writes"}, tc.args...)...)

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) || len(h.writes) != 0 {
					t.Fatalf("move %v = %v, writes %v; want an error containing %q and no write", tc.args, err, h.writes, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(h.writes, tc.wantWrites) {
				t.Errorf("writes = %v, want %v", h.writes, tc.wantWrites)
			}
		})
	}
}

func TestMonthsMovePrintsBothRowsAndPreviewsADryRun(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "months", "move", "10", "--from", "Visa", "--to", "Rent", "--allow-writes")
	if err != nil {
		t.Fatal(err)
	}

	want := `NAME   ASSIGNED    ACTIVITY  AVAILABLE  TARGET        UNDERFUNDED
Visa     $87.81      $97.81     $87.81
Rent  $1,510.00  -$1,500.00     $10.00  MF $1,500.00        $0.00
moved $10.00 from Credit Card Payments: Visa to Bills: Rent in 2026-09
`
	if out != want {
		t.Errorf("move =\n%s\nwant\n%s", out, want)
	}

	// The dry run applies the arithmetic itself: assigned and available
	// move by the amount, and the hidden target's underfunded amount
	// shrinks to zero.
	dry, err := h.execute(t, "months", "move", "100", "--from", "Visa", "--to", "Wishes: Internet", "--force", "--dry-run", "--csv")
	if err != nil {
		t.Fatal(err)
	}
	wantDry := `id,name,group,group_id,month,hidden,internal,note,assigned,activity,available,target_type,target_amount,target_date,target_cadence,target_cadence_frequency,target_day,target_needs_whole_amount,target_created_month,target_percent_complete,target_months_to_assign,target_underfunded,target_overall_funded,target_overall_left,target_snoozed_at,dry_run
c6,Visa,Credit Card Payments,g4,2026-09,false,false,,-2.19,97.81,-2.19,,,,,,,,,,,,,,,true
c5,Internet,Wishes,g3,2026-09,true,false,,100.00,0.00,200.00,TBD,500.00,2027-01-01,,,,,2026-05,20,4,0.00,100.00,400.00,,true
`
	if dry != wantDry {
		t.Errorf("move --dry-run --csv =\n%s\nwant\n%s", dry, wantDry)
	}
	if len(h.writes) != 2 {
		t.Errorf("writes = %v, want only the real run's two", h.writes)
	}
}

func TestMonthsMovePartialFailurePrintsTheReversal(t *testing.T) {
	h := newHarness(t)
	h.failWrite = "/categories/c2"

	out, err := h.execute(t, "months", "move", "10", "--from", "Visa", "--to", "Rent", "--allow-writes")

	if err == nil || ExitCode(err) != ExitFailure {
		t.Fatalf("move with a failing second write = %v (exit %d), want exit 1", err, ExitCode(err))
	}
	for _, want := range []string{
		"write 2 of 2, Bills: Rent:",
		`1 write already applied, reverse with: ynab months assign c6 10.00 --add --plan p1 --month 2026-09 --allow-writes`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
	wantOut := `NAME  ASSIGNED  ACTIVITY  AVAILABLE  TARGET  UNDERFUNDED
Visa    $87.81    $97.81     $87.81
1 write of 2 applied before the failure
`
	if out != wantOut {
		t.Errorf("stdout after the failure =\n%s\nwant\n%s", out, wantOut)
	}
	if len(h.writes) != 2 {
		t.Errorf("writes = %v, want the source write and the failed target write", h.writes)
	}

	h = newHarness(t)
	h.failWrite = "/categories/c6"
	out, err = h.execute(t, "months", "move", "10", "--from", "Visa", "--to", "Rent", "--allow-writes")
	if err == nil || !strings.Contains(err.Error(), "write 1 of 2, Credit Card Payments: Visa:") || strings.Contains(err.Error(), "reverse") || out != "" {
		t.Errorf("move with a failing first write = %q, %v; want the write named, no reversal, and empty stdout", out, err)
	}
}

func TestMonthsCoverMovesExactlyTheShortfall(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantWrites []string
		wantOut    string
		wantErr    string
	}{
		{
			"from a category", []string{"Dining Out", "--from", "Visa"},
			[]string{`PATCH /plans/p1/months/2026-09-01/categories/c6 {"category":{"budgeted":72810}}`, `PATCH /plans/p1/months/2026-09-01/categories/c3 {"category":{"budgeted":225000}}`},
			"covered Fun: Dining Out with $25.00 from Credit Card Payments: Visa in 2026-09\n", "",
		},
		{
			"from ready to assign forced", []string{"Dining Out", "--from", "ready-to-assign", "--force"},
			[]string{`PATCH /plans/p1/months/2026-09-01/categories/c3 {"category":{"budgeted":225000}}`},
			"covered Fun: Dining Out with $25.00 from ready to assign in 2026-09\n", "",
		},
		{"nothing to cover", []string{"Rent", "--from", "Visa"}, nil, "nothing to cover: Bills: Rent has $0.00 available in 2026-09\n", ""},
		{"from ready to assign below the shortfall", []string{"Dining Out", "--from", "ready-to-assign"}, nil, "", "ready to assign has -$500.00 available in 2026-09, less than $25.00"},
		{"from itself", []string{"Dining Out", "--from", "Fun: Dining Out"}, nil, "", "the category to cover"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)

			out, err := h.execute(t, append([]string{"months", "cover", "--allow-writes"}, tc.args...)...)

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) || len(h.writes) != 0 {
					t.Fatalf("cover %v = %v, writes %v; want an error containing %q and no write", tc.args, err, h.writes, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(h.writes, tc.wantWrites) {
				t.Errorf("writes = %v, want %v", h.writes, tc.wantWrites)
			}
			if !strings.HasSuffix(out, tc.wantOut) {
				t.Errorf("cover output =\n%s\nwant it to end with %q", out, tc.wantOut)
			}
		})
	}

	h := newHarness(t)
	out, stderr, err := h.executeStreams(t, "months", "cover", "Rent", "--from", "Visa", "--allow-writes", "--jsonl")
	if err != nil || out != "" || stderr != "nothing to cover: Bills: Rent has $0.00 available in 2026-09\n" {
		t.Errorf("cover with nothing to cover --jsonl = %q, stderr %q, %v; want empty stdout and the line on stderr", out, stderr, err)
	}
}

func TestMonthsFundAssignsWhatTargetsNeed(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantWrites []string
		wantOut    string
		wantErr    string
	}{
		{
			// Only the hidden Wishes: Internet is underfunded, so the
			// sweep finds nothing and proves hidden categories stay out.
			"all underfunded skips hidden", []string{"--all-underfunded"}, nil, "nothing to fund in 2026-09\n", "",
		},
		{
			"named from a source forced", []string{"Wishes: Internet", "Bills: Internet", "--from", "Visa", "--force"},
			[]string{`PATCH /plans/p1/months/2026-09-01/categories/c6 {"category":{"budgeted":-2190}}`, `PATCH /plans/p1/months/2026-09-01/categories/c5 {"category":{"budgeted":100000}}`},
			"funded 1 category with $100.00 from Credit Card Payments: Visa in 2026-09, 1 category already funded\n", "",
		},
		{
			"named from ready to assign forced", []string{"Wishes: Internet", "--force"},
			[]string{`PATCH /plans/p1/months/2026-09-01/categories/c5 {"category":{"budgeted":100000}}`},
			"funded 1 category with $100.00 from ready to assign in 2026-09\n", "",
		},
		{"all named already funded", []string{"Rent", "Bills: Internet"}, nil, "nothing to fund in 2026-09, 2 categories already funded\n", ""},
		{
			// The same category named by ID and by name is funded once.
			"named twice forced", []string{"c5", "Wishes: Internet", "--from", "Visa", "--force"},
			[]string{`PATCH /plans/p1/months/2026-09-01/categories/c6 {"category":{"budgeted":-2190}}`, `PATCH /plans/p1/months/2026-09-01/categories/c5 {"category":{"budgeted":100000}}`},
			"funded 1 category with $100.00 from Credit Card Payments: Visa in 2026-09\n", "",
		},
		{"source below the total", []string{"Wishes: Internet", "--from", "Visa"}, nil, "", "Credit Card Payments: Visa has $97.81 available in 2026-09, less than $100.00"},
		{"ready to assign below the total", []string{"Wishes: Internet"}, nil, "", "ready to assign has -$500.00 available in 2026-09, less than $100.00"},
		{"no target", []string{"Dining Out"}, nil, "", `category "Dining Out" has no target to fund`},
		{"source among the targets", []string{"Wishes: Internet", "--from", "Wishes: Internet"}, nil, "", "one of the categories to fund"},
		{"names and the sweep", []string{"Rent", "--all-underfunded"}, nil, "", "not both"},
		{"neither names nor the sweep", nil, nil, "", "at least one category"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)

			out, err := h.execute(t, append([]string{"months", "fund", "--allow-writes"}, tc.args...)...)

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) || len(h.writes) != 0 {
					t.Fatalf("fund %v = %v, writes %v; want an error containing %q and no write", tc.args, err, h.writes, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(h.writes, tc.wantWrites) {
				t.Errorf("writes = %v, want %v", h.writes, tc.wantWrites)
			}
			if !strings.HasSuffix(out, tc.wantOut) {
				t.Errorf("fund output =\n%s\nwant it to end with %q", out, tc.wantOut)
			}
		})
	}
}

func TestMonthsFundPrintsTheSourceThenTheFundedRows(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "months", "fund", "Wishes: Internet", "--from", "Visa", "--force", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}

	want := `NAME               ASSIGNED  ACTIVITY  AVAILABLE  TARGET       UNDERFUNDED
Visa                 -$2.19    $97.81     -$2.19
Internet (hidden)   $100.00     $0.00    $200.00  TBD $500.00        $0.00
dry run, nothing changed: would fund 1 category with $100.00 from Credit Card Payments: Visa in 2026-09
`
	if out != want {
		t.Errorf("fund --dry-run =\n%s\nwant\n%s", out, want)
	}
	if len(h.writes) != 0 {
		t.Errorf("dry run wrote: %v", h.writes)
	}
}
