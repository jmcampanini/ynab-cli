package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

func TestReportsFundingListsTargetsInPlanOrder(t *testing.T) {
	h := newHarness(t)

	out, err := h.execute(t, "reports", "funding")
	if err != nil {
		t.Fatal(err)
	}
	jsonl, err := h.execute(t, "reports", "funding", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}

	want := `GROUP  NAME      TARGET TYPE            TARGET   ASSIGNED    ACTIVITY  AVAILABLE  UNDERFUNDED  COMPLETE  MONTHS LEFT
Bills  Internet  plan your spending     $80.00     $80.00     -$79.99      $0.01        $0.00      100%            1
Bills  Rent      monthly funding     $1,500.00  $1,500.00  -$1,500.00      $0.00        $0.00      100%            1
2 targets in 2026-09, 0 underfunded, $0.00 still needed
`
	if out != want {
		t.Errorf("reports funding =\n%s\nwant\n%s", out, want)
	}
	wantJSONL := `{"id":"c1","name":"Internet","group":"Bills","month":"2026-09","target_type":"NEED","target_amount":80.00,"assigned":80.00,"activity":-79.99,"available":0.01,"underfunded":0.00,"percent_complete":100,"months_to_assign":1}
{"id":"c2","name":"Rent","group":"Bills","month":"2026-09","target_type":"MF","target_amount":1500.00,"assigned":1500.00,"activity":-1500.00,"available":0.00,"underfunded":0.00,"percent_complete":100,"months_to_assign":1}
`
	if jsonl != wantJSONL {
		t.Errorf("reports funding --jsonl =\n%s\nwant\n%s", jsonl, wantJSONL)
	}
	if want := []string{"/plans", "/plans/p1/months/2026-09-01", "/plans/p1/categories"}; !reflect.DeepEqual(h.requests[:3], want) {
		t.Errorf("requests = %v, want %v first", h.requests, want)
	}
}

func TestFundingReportKeepsUnderfundedVisibleTargets(t *testing.T) {
	need, monthly := "NEED", "MF"
	short, none := ynab.Amount(100000), ynab.Amount(0)
	groups := []ynab.CategoryGroup{
		{ID: "g", Name: "Bills", Categories: []ynab.Category{
			{ID: "a", Name: "Internet", TargetType: &need, TargetAmount: 80000, TargetUnderfunded: &short},
			{ID: "b", Name: "Rent", TargetType: &monthly, TargetAmount: 1500000, TargetUnderfunded: &none},
			{ID: "c", Name: "Fun"},
			{ID: "d", Name: "Old", Hidden: true, TargetType: &need, TargetUnderfunded: &short},
		}},
		{ID: "m", Name: "Internal Master Category", Internal: true, Categories: []ynab.Category{
			{ID: "e", Name: "Inflow: Ready to Assign", Internal: true, TargetType: &need, TargetUnderfunded: &short},
		}},
	}
	// The month rows repeat the names and flags, as the API's do.
	month := ynab.Month{Month: "2026-09-01", Categories: []ynab.Category{
		{ID: "a", Name: "Internet", Assigned: 20000, TargetType: &need, TargetAmount: 80000, TargetUnderfunded: &short},
		{ID: "b", Name: "Rent", Assigned: 1500000, TargetType: &monthly, TargetAmount: 1500000, TargetUnderfunded: &none},
		{ID: "c", Name: "Fun"},
		{ID: "d", Name: "Old", Hidden: true, TargetType: &need, TargetUnderfunded: &short},
		{ID: "e", Name: "Inflow: Ready to Assign", Internal: true, TargetType: &need, TargetUnderfunded: &short},
	}}

	all, err := newFundingReport(groups, month, false)
	if err != nil {
		t.Fatal(err)
	}
	only, err := newFundingReport(groups, month, true)
	if err != nil {
		t.Fatal(err)
	}

	if names := recordNames(all.records); !reflect.DeepEqual(names, []string{"Internet", "Rent"}) {
		t.Errorf("all rows = %v, want Internet and Rent", names)
	}
	if names := recordNames(only.records); !reflect.DeepEqual(names, []string{"Internet"}) {
		t.Errorf("--underfunded rows = %v, want Internet", names)
	}
	for _, report := range []fundingReport{all, only} {
		if report.underfundedCount != 1 || report.underfundedTotal != short {
			t.Errorf("summary = %d underfunded totaling %s, want 1 totaling %s", report.underfundedCount, report.underfundedTotal, short)
		}
	}
	if all.records[0].Assigned != 20000 {
		t.Errorf("Internet assigned = %s, want the month's 20.00", all.records[0].Assigned)
	}
}

func recordNames(records []fundingRecord) []string {
	names := make([]string, len(records))
	for i, record := range records {
		names[i] = record.Name
	}
	return names
}

func TestReportsSpendingTotalsByCategoryAndPayee(t *testing.T) {
	h := newHarness(t)

	byCategory, err := h.execute(t, "reports", "spending")
	if err != nil {
		t.Fatal(err)
	}
	categoryRequests := h.requests
	byPayee, err := h.execute(t, "reports", "spending", "--by", "payee")
	if err != nil {
		t.Fatal(err)
	}
	jsonl, err := h.execute(t, "reports", "spending", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}

	// The transfer pair t2 and t2b is left out; the splits t1 and t8
	// count per line, and t8's lines carry their own payees.
	wantCategory := `GROUP  CATEGORY       OUTFLOWS  INFLOWS      NET  TRANSACTIONS
Fun    Dining Out      -$82.34    $0.00  -$82.34             3
Bills  Internet        -$55.00    $0.00  -$55.00             2
       Uncategorized   -$25.00    $0.00  -$25.00             2
3 categories, 5 transactions in 2026-09: outflows -$162.34, inflows $0.00, net -$162.34
`
	if byCategory != wantCategory {
		t.Errorf("reports spending =\n%s\nwant\n%s", byCategory, wantCategory)
	}
	wantPayee := `PAYEE           OUTFLOWS  INFLOWS       NET  TRANSACTIONS
Costco          -$100.00    $0.00  -$100.00             1
Amazon           -$22.34    $0.00   -$22.34             2
New Shop         -$20.00    $0.00   -$20.00             1
Unknown Vendor   -$20.00    $0.00   -$20.00             2
4 payees, 5 transactions in 2026-09: outflows -$162.34, inflows $0.00, net -$162.34
`
	if byPayee != wantPayee {
		t.Errorf("reports spending --by payee =\n%s\nwant\n%s", byPayee, wantPayee)
	}
	wantJSONL := `{"id":"c3","name":"Dining Out","group":"Fun","outflows":-82.34,"inflows":0.00,"net":-82.34,"transactions":3}
{"id":"c1","name":"Internet","group":"Bills","outflows":-55.00,"inflows":0.00,"net":-55.00,"transactions":2}
{"name":"Uncategorized","outflows":-25.00,"inflows":0.00,"net":-25.00,"transactions":2}
`
	if jsonl != wantJSONL {
		t.Errorf("reports spending --jsonl =\n%s\nwant\n%s", jsonl, wantJSONL)
	}
	if want := []string{"/plans", "/plans/p1/accounts", "/plans/p1/categories", "/plans/p1/months/2026-09-01/transactions"}; !reflect.DeepEqual(categoryRequests, want) {
		t.Errorf("requests = %v, want %v", categoryRequests, want)
	}
}

func TestReportsSpendingRanges(t *testing.T) {
	h := newHarness(t)

	since, err := h.execute(t, "reports", "spending", "--since", "2026-08-01", "--csv")
	if err != nil {
		t.Fatal(err)
	}
	sinceRequest := h.requests[len(h.requests)-1]
	account, err := h.execute(t, "reports", "spending", "--account", "Visa", "--month", "2026-09")
	if err != nil {
		t.Fatal(err)
	}

	wantSince := `id,name,group,outflows,inflows,net,transactions
c2,Rent,Bills,-1500.00,0.00,-1500.00,1
c3,Dining Out,Fun,-82.34,0.00,-82.34,3
c1,Internet,Bills,-55.00,0.00,-55.00,2
,Uncategorized,,-25.00,0.00,-25.00,2
c7,Inflow: Ready to Assign,Internal Master Category,0.00,3000.00,3000.00,1
`
	if since != wantSince {
		t.Errorf("reports spending --since --csv =\n%s\nwant\n%s", since, wantSince)
	}
	if sinceRequest != "/plans/p1/transactions?since_date=2026-08-01" {
		t.Errorf("--since listing = %s, want the plan listing since 2026-08-01", sinceRequest)
	}
	if !strings.Contains(account, "Dining Out   -$12.34") || !strings.Contains(account, "1 category, 1 transaction in 2026-09") {
		t.Errorf("reports spending --account Visa =\n%s", account)
	}
}

func TestSpendingLinesExcludeOnPlanTransfers(t *testing.T) {
	onPlan := map[string]bool{"checking": true, "visa": true, "brokerage": false}
	checking, visa, brokerage, groceries := "checking", "visa", "brokerage", "groceries"
	cases := []struct {
		name string
		tx   ynab.Transaction
		want []ynab.Amount
	}{
		{"transfer between on-plan accounts", ynab.Transaction{AccountID: "checking", Amount: -50000, TransferAccountID: &visa}, nil},
		{"transfer to a tracking account", ynab.Transaction{AccountID: "checking", Amount: -50000, TransferAccountID: &brokerage}, []ynab.Amount{-50000}},
		{"the tracking side of that transfer", ynab.Transaction{AccountID: "brokerage", Amount: 50000, TransferAccountID: &checking}, nil},
		{"a tracking account's own transaction", ynab.Transaction{AccountID: "brokerage", Amount: -1000}, nil},
		{"split with a transfer line", ynab.Transaction{AccountID: "checking", Amount: -80000, Subtransactions: []ynab.Subtransaction{
			{Amount: -30000, CategoryID: &groceries}, {Amount: -50000, TransferAccountID: &visa},
		}}, []ynab.Amount{-30000}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := spendingLines(tc.tx, onPlan)

			var got []ynab.Amount
			for _, line := range lines {
				got = append(got, line.amount)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("spendingLines amounts = %v, want %v", got, tc.want)
			}
		})
	}
}
