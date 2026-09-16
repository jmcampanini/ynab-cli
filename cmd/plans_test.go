package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

func TestPlansListMarksConfiguredPlan(t *testing.T) {
	h := newHarness(t)

	table, err := h.execute(t, "plans", "list")
	if err != nil {
		t.Fatal(err)
	}
	jsonl, err := h.execute(t, "plans", "list", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}
	csv, err := h.execute(t, "plans", "list", "--csv")
	if err != nil {
		t.Fatal(err)
	}

	wantTable := `   NAME           ID  LAST MODIFIED        FIRST MONTH  LAST MONTH
*  Household      p1  2026-09-01 12:00:00  2024-01      2026-10
   Side Business  p2  2026-08-01 12:00:00  2025-01      2026-09
2 plans
`
	if table != wantTable {
		t.Errorf("table =\n%s\nwant\n%s", table, wantTable)
	}
	wantJSONL := `{"id":"p1","name":"Household","first_month":"2024-01","last_month":"2026-10","last_modified_on":"2026-09-01T12:00:00Z","currency":"USD","date_format":"MM/DD/YYYY","configured":true}
{"id":"p2","name":"Side Business","first_month":"2025-01","last_month":"2026-09","last_modified_on":"2026-08-01T12:00:00Z","configured":false}
`
	if jsonl != wantJSONL {
		t.Errorf("jsonl =\n%s\nwant\n%s", jsonl, wantJSONL)
	}
	wantCSV := `id,name,first_month,last_month,last_modified_on,currency,date_format,configured
p1,Household,2024-01,2026-10,2026-09-01T12:00:00Z,USD,MM/DD/YYYY,true
p2,Side Business,2025-01,2026-09,2026-08-01T12:00:00Z,,,false
`
	if csv != wantCSV {
		t.Errorf("csv =\n%s\nwant\n%s", csv, wantCSV)
	}
}

func TestPlansListNeedsNoPlan(t *testing.T) {
	h := newHarness(t)
	t.Setenv("YNAB_PLAN", "")

	out, err := h.execute(t, "plans", "list")

	if err != nil || strings.Contains(out, "*") {
		t.Errorf("plans list = %q, %v; want no marker and no error", out, err)
	}
}

func TestPlansGetResolvesByIDOrName(t *testing.T) {
	h := newHarness(t)

	byName, err := h.execute(t, "plans", "get")
	if err != nil {
		t.Fatal(err)
	}
	byID, err := h.execute(t, "plans", "get", "--plan", "p1", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}

	want := `name           Household
id             p1
first month    2024-01
last month     2026-10
last modified  2026-09-01 12:00:00
currency       USD (123,456.78)
date format    MM/DD/YYYY
`
	if byName != want {
		t.Errorf("plans get =\n%s\nwant\n%s", byName, want)
	}
	if !strings.HasPrefix(byID, `{"id":"p1","name":"Household"`) || strings.Count(byID, "\n") != 1 {
		t.Errorf("plans get --jsonl = %q", byID)
	}
	if len(h.requests) != 2 || h.requests[0] != "/plans" {
		t.Errorf("requests = %v, want one plans request per invocation", h.requests)
	}
}

func TestPlanMissAndAmbiguityListNames(t *testing.T) {
	h := newHarness(t)

	_, err := h.execute(t, "plans", "get", "--plan", "Nope")
	if err == nil || !strings.Contains(err.Error(), `"Household", "Side Business"`) || ExitCode(err) != ExitFailure {
		t.Errorf("plans get with unknown plan = %v", err)
	}

	twins := []ynab.Plan{{ID: "x", Name: "Twin"}, {ID: "y", Name: "twin"}}
	if got := matchPlan("TWIN", twins); len(got) != 2 {
		t.Errorf("matchPlan(TWIN) = %v, want both plans so the caller can report ambiguity", got)
	}
	if got := matchPlan("y", twins); len(got) != 1 || got[0].ID != "y" {
		t.Errorf("matchPlan(y) = %v, want the ID match alone", got)
	}
}

func TestPlansStatusSummarizesTheCurrentMonth(t *testing.T) {
	h := newHarness(t)

	table, err := h.execute(t, "plans", "status")
	if err != nil {
		t.Fatal(err)
	}
	jsonl, err := h.execute(t, "plans", "status", "--jsonl")
	if err != nil {
		t.Fatal(err)
	}

	wantTable := `month            2026-09
ready to assign  -$500.00
age of money
overspent        1 category, -$25.00: Dining Out -$25.00
underfunded      none
unapproved       3 transactions
uncategorized    4 transactions
import errors    Visa
`
	if table != wantTable {
		t.Errorf("plans status =\n%s\nwant\n%s", table, wantTable)
	}
	wantJSONL := `{"month":"2026-09","ready_to_assign":-500.00,"overspent_count":1,"overspent_total":-25.00,"underfunded_count":0,"underfunded_total":0.00,"unapproved_count":3,"uncategorized_count":4,"import_error_accounts":["Visa"]}` + "\n"
	if jsonl != wantJSONL {
		t.Errorf("plans status --jsonl =\n%s\nwant\n%s", jsonl, wantJSONL)
	}
	want := []string{
		"/plans", "/plans/p1/months/2026-09-01", "/plans/p1/categories", "/plans/p1/accounts",
		"/plans/p1/transactions?type=unapproved", "/plans/p1/transactions?type=uncategorized",
	}
	if !reflect.DeepEqual(h.requests[:6], want) {
		t.Errorf("requests = %v, want %v", h.requests[:6], want)
	}
}

func TestPlanStatusUnderfundedAndImportErrorsFromFixtures(t *testing.T) {
	underfunded := ynab.Amount(100000)
	targetType := "NEED"
	groups := []ynab.CategoryGroup{{ID: "g", Name: "Bills", Categories: []ynab.Category{
		{ID: "c", Name: "Internet", GroupID: "g", Available: 5000, TargetType: &targetType, TargetUnderfunded: &underfunded},
		{ID: "h", Name: "Hidden", GroupID: "g", Hidden: true, Available: -1000, TargetType: &targetType, TargetUnderfunded: &underfunded},
	}}}
	month := ynab.Month{Month: "2026-09-01", Categories: groups[0].Categories}
	accounts := []ynab.Account{{Name: "Open", DirectImportInError: true}, {Name: "Closed", Closed: true, DirectImportInError: true}}

	status, err := newPlanStatus(month, groups, accounts, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	want := statusRecord{Month: "2026-09", UnderfundedCount: 1, UnderfundedTotal: 100000, ImportErrorAccounts: []string{"Open"}}
	if !reflect.DeepEqual(status.record, want) {
		t.Errorf("status = %+v, want %+v", status.record, want)
	}
	if got := status.fields(nil)[4]; got != [2]string{"underfunded", "1 target, 100.00: Internet 100.00"} {
		t.Errorf("underfunded field = %q", got)
	}
}
