package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// fixture is a fake YNAB API with two plans; plan "p1" has three accounts,
// five category groups, and two months. Plan "p2" exists so a name lookup
// can be exercised against several plans. The clock is fixed in September
// 2026, so "current" is 2026-09.
const (
	fixturePlans = `{"data":{"plans":[
{"id":"p1","name":"Household","last_modified_on":"2026-09-01T12:00:00Z","first_month":"2024-01-01","last_month":"2026-10-01",
 "date_format":{"format":"MM/DD/YYYY"},
 "currency_format":{"iso_code":"USD","example_format":"123,456.78","decimal_digits":2,"decimal_separator":".","symbol_first":true,"group_separator":",","currency_symbol":"$","display_symbol":true}},
{"id":"p2","name":"Side Business","last_modified_on":"2026-08-01T12:00:00Z","first_month":"2025-01-01","last_month":"2026-09-01","date_format":null,"currency_format":null}
]}}`
	fixtureAccounts = `{"data":{"accounts":[
{"id":"a1","name":"Chase Checking","type":"checking","on_budget":true,"closed":false,"note":"Main","balance":1234560,"cleared_balance":1000000,"uncleared_balance":234560,"transfer_payee_id":"tp1","direct_import_linked":true,"direct_import_in_error":false,"last_reconciled_at":"2026-08-31T00:00:00Z","deleted":false},
{"id":"a2","name":"Visa","type":"creditCard","on_budget":true,"closed":false,"note":null,"balance":-97810,"cleared_balance":-97810,"uncleared_balance":0,"transfer_payee_id":"tp2","direct_import_linked":true,"direct_import_in_error":true,"last_reconciled_at":null,"deleted":false},
{"id":"a3","name":"Old Savings","type":"savings","on_budget":false,"closed":true,"note":null,"balance":0,"cleared_balance":0,"uncleared_balance":0,"transfer_payee_id":null,"direct_import_linked":false,"direct_import_in_error":false,"last_reconciled_at":null,"deleted":false}
]}}`
	// fixtureCategoryGroups carries September's amounts: Dining Out is
	// overspent, Old Hobby is hidden, Wishes is a hidden group whose
	// Internet shares its name with Bills' Internet, and the two internal
	// groups hold a credit card payment and the inflow category.
	fixtureCategoryGroups = `{"data":{"category_groups":[
{"id":"g1","name":"Bills","hidden":false,"internal":false,"deleted":false,"categories":[
 {"id":"c1","category_group_id":"g1","category_group_name":"Bills","name":"Internet","hidden":false,"internal":false,"note":"Fiber","budgeted":80000,"activity":-79990,"balance":10,"goal_type":"NEED","goal_needs_whole_amount":true,"goal_day":null,"goal_cadence":1,"goal_cadence_frequency":1,"goal_creation_month":"2025-08-01","goal_target":80000,"goal_target_date":null,"goal_percentage_complete":100,"goal_months_to_budget":1,"goal_under_funded":0,"goal_overall_funded":80000,"goal_overall_left":0,"goal_snoozed_at":null,"deleted":false},
 {"id":"c2","category_group_id":"g1","category_group_name":"Bills","name":"Rent","hidden":false,"internal":false,"note":null,"budgeted":1500000,"activity":-1500000,"balance":0,"goal_type":"MF","goal_needs_whole_amount":null,"goal_day":null,"goal_cadence":1,"goal_cadence_frequency":1,"goal_creation_month":"2024-01-01","goal_target":1500000,"goal_target_date":null,"goal_percentage_complete":100,"goal_months_to_budget":1,"goal_under_funded":0,"goal_overall_funded":1500000,"goal_overall_left":0,"goal_snoozed_at":null,"deleted":false}]},
{"id":"g2","name":"Fun","hidden":false,"internal":false,"deleted":false,"categories":[
 {"id":"c3","category_group_id":"g2","category_group_name":"Fun","name":"Dining Out","hidden":false,"internal":false,"note":null,"budgeted":200000,"activity":-225000,"balance":-25000,"goal_type":null,"goal_needs_whole_amount":null,"goal_day":null,"goal_cadence":null,"goal_cadence_frequency":null,"goal_creation_month":null,"goal_target":0,"goal_target_date":null,"goal_percentage_complete":null,"goal_months_to_budget":null,"goal_under_funded":null,"goal_overall_funded":null,"goal_overall_left":null,"goal_snoozed_at":null,"deleted":false},
 {"id":"c4","category_group_id":"g2","category_group_name":"Fun","name":"Old Hobby","hidden":true,"internal":false,"note":null,"budgeted":0,"activity":0,"balance":5000,"goal_type":null,"goal_needs_whole_amount":null,"goal_day":null,"goal_cadence":null,"goal_cadence_frequency":null,"goal_creation_month":null,"goal_target":0,"goal_target_date":null,"goal_percentage_complete":null,"goal_months_to_budget":null,"goal_under_funded":null,"goal_overall_funded":null,"goal_overall_left":null,"goal_snoozed_at":null,"deleted":false}]},
{"id":"g3","name":"Wishes","hidden":true,"internal":false,"deleted":false,"categories":[
 {"id":"c5","category_group_id":"g3","category_group_name":"Wishes","name":"Internet","hidden":true,"internal":false,"note":null,"budgeted":0,"activity":0,"balance":100000,"goal_type":"TBD","goal_needs_whole_amount":null,"goal_day":null,"goal_cadence":null,"goal_cadence_frequency":null,"goal_creation_month":"2026-05-01","goal_target":500000,"goal_target_date":"2027-01-01","goal_percentage_complete":20,"goal_months_to_budget":4,"goal_under_funded":100000,"goal_overall_funded":100000,"goal_overall_left":400000,"goal_snoozed_at":null,"deleted":false}]},
{"id":"g4","name":"Credit Card Payments","hidden":false,"internal":true,"deleted":false,"categories":[
 {"id":"c6","category_group_id":"g4","category_group_name":"Credit Card Payments","name":"Visa","hidden":false,"internal":false,"note":null,"budgeted":97810,"activity":97810,"balance":97810,"goal_type":null,"goal_needs_whole_amount":null,"goal_day":null,"goal_cadence":null,"goal_cadence_frequency":null,"goal_creation_month":null,"goal_target":0,"goal_target_date":null,"goal_percentage_complete":null,"goal_months_to_budget":null,"goal_under_funded":null,"goal_overall_funded":null,"goal_overall_left":null,"goal_snoozed_at":null,"deleted":false}]},
{"id":"g5","name":"Internal Master Category","hidden":false,"internal":true,"deleted":false,"categories":[
 {"id":"c7","category_group_id":"g5","category_group_name":"Internal Master Category","name":"Inflow: Ready to Assign","hidden":false,"internal":true,"note":null,"budgeted":0,"activity":0,"balance":0,"goal_type":null,"goal_needs_whole_amount":null,"goal_day":null,"goal_cadence":null,"goal_cadence_frequency":null,"goal_creation_month":null,"goal_target":0,"goal_target_date":null,"goal_percentage_complete":null,"goal_months_to_budget":null,"goal_under_funded":null,"goal_overall_funded":null,"goal_overall_left":null,"goal_snoozed_at":null,"deleted":false}]}
]}}`
	// fixtureAugustCategories are the same categories with August's
	// amounts, in an order unlike the groups' to prove grouping does not
	// rely on the month endpoint's order.
	fixtureAugustCategories = `[
{"id":"c3","category_group_id":"g2","category_group_name":"Fun","name":"Dining Out","hidden":false,"internal":false,"note":null,"budgeted":200000,"activity":-180000,"balance":20000,"goal_type":null,"goal_target":0,"deleted":false},
{"id":"c1","category_group_id":"g1","category_group_name":"Bills","name":"Internet","hidden":false,"internal":false,"note":"Fiber","budgeted":80000,"activity":-80000,"balance":0,"goal_type":"NEED","goal_needs_whole_amount":true,"goal_day":null,"goal_cadence":1,"goal_cadence_frequency":1,"goal_creation_month":"2025-08-01","goal_target":80000,"goal_target_date":null,"goal_percentage_complete":100,"goal_months_to_budget":1,"goal_under_funded":0,"goal_overall_funded":80000,"goal_overall_left":0,"goal_snoozed_at":null,"deleted":false},
{"id":"c2","category_group_id":"g1","category_group_name":"Bills","name":"Rent","hidden":false,"internal":false,"note":null,"budgeted":1500000,"activity":-1500000,"balance":0,"goal_type":"MF","goal_cadence":1,"goal_cadence_frequency":1,"goal_creation_month":"2024-01-01","goal_target":1500000,"goal_percentage_complete":100,"goal_months_to_budget":1,"goal_under_funded":0,"goal_overall_funded":1500000,"goal_overall_left":0,"deleted":false},
{"id":"c4","category_group_id":"g2","category_group_name":"Fun","name":"Old Hobby","hidden":true,"internal":false,"note":null,"budgeted":0,"activity":0,"balance":5000,"goal_type":null,"goal_target":0,"deleted":false},
{"id":"c5","category_group_id":"g3","category_group_name":"Wishes","name":"Internet","hidden":true,"internal":false,"note":null,"budgeted":0,"activity":0,"balance":100000,"goal_type":"TBD","goal_creation_month":"2026-05-01","goal_target":500000,"goal_target_date":"2027-01-01","goal_percentage_complete":20,"goal_months_to_budget":5,"goal_under_funded":100000,"goal_overall_funded":100000,"goal_overall_left":400000,"deleted":false},
{"id":"c6","category_group_id":"g4","category_group_name":"Credit Card Payments","name":"Visa","hidden":false,"internal":false,"note":null,"budgeted":50000,"activity":50000,"balance":50000,"goal_type":null,"goal_target":0,"deleted":false},
{"id":"c7","category_group_id":"g5","category_group_name":"Internal Master Category","name":"Inflow: Ready to Assign","hidden":false,"internal":true,"note":null,"budgeted":0,"activity":0,"balance":0,"goal_type":null,"goal_target":0,"deleted":false}
]`
	fixtureAugustTotals    = `"month":"2026-08-01","note":"August","income":3000000,"budgeted":1780000,"activity":-1710000,"to_be_budgeted":1220000,"age_of_money":30,"deleted":false`
	fixtureSeptemberTotals = `"month":"2026-09-01","note":null,"income":0,"budgeted":1780000,"activity":-1707180,"to_be_budgeted":-500000,"age_of_money":null,"deleted":false`
)

func notFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = io.WriteString(w, `{"error":{"id":"404.2","name":"resource_not_found","detail":"Resource not found"}}`)
}

// fixtureNow is the harness clock.
var fixtureNow = time.Date(2026, time.September, 15, 12, 0, 0, 0, time.UTC)

// fixtureCurrentCategories flattens the group fixture into the month
// endpoint's shape so September's month detail carries the same amounts
// as the categories endpoint.
func fixtureCurrentCategories(t *testing.T) string {
	t.Helper()
	var groups struct {
		Data struct {
			Groups []struct {
				Categories []json.RawMessage `json:"categories"`
			} `json:"category_groups"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(fixtureCategoryGroups), &groups); err != nil {
		t.Fatal(err)
	}
	var categories []json.RawMessage
	for _, group := range groups.Data.Groups {
		categories = append(categories, group.Categories...)
	}
	flattened, err := json.Marshal(categories)
	if err != nil {
		t.Fatal(err)
	}
	return string(flattened)
}

// fixtureCategoryIn returns one category of a month fixture array by ID.
func fixtureCategoryIn(t *testing.T, categories, id string) (string, bool) {
	t.Helper()
	var entries []json.RawMessage
	if err := json.Unmarshal([]byte(categories), &entries); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		var header struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(entry, &header); err != nil {
			t.Fatal(err)
		}
		if header.ID == id {
			return string(entry), true
		}
	}
	return "", false
}

// harness runs a fresh root against a fake API in an isolated environment
// and records the API paths each execution requested.
type harness struct {
	deps     dependencies
	requests []string
	terminal bool
	env      map[string]string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{env: map[string]string{}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.requests = append(h.requests, r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer good-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error":{"id":"401","name":"unauthorized","detail":"Unauthorized"}}`)
			return
		}
		months := map[string]string{"2026-08-01": fixtureAugustTotals, "2026-09-01": fixtureSeptemberTotals}
		monthCategories := map[string]string{"2026-08-01": fixtureAugustCategories, "2026-09-01": fixtureCurrentCategories(t)}
		monthPath := regexp.MustCompile(`^/plans/p1/months/(\d{4}-\d{2}-\d{2})(?:/categories/(\w+))?$`)
		switch match := monthPath.FindStringSubmatch(r.URL.Path); {
		case r.URL.Path == "/plans":
			_, _ = io.WriteString(w, fixturePlans)
		case r.URL.Path == "/plans/p1/accounts":
			_, _ = io.WriteString(w, fixtureAccounts)
		case r.URL.Path == "/plans/p1/categories":
			_, _ = io.WriteString(w, fixtureCategoryGroups)
		case r.URL.Path == "/plans/p1/months":
			_, _ = io.WriteString(w, `{"data":{"months":[{`+fixtureAugustTotals+`},{`+fixtureSeptemberTotals+`}]}}`)
		case match != nil && match[2] == "" && months[match[1]] != "":
			_, _ = io.WriteString(w, `{"data":{"month":{`+months[match[1]]+`,"categories":`+monthCategories[match[1]]+`}}}`)
		case match != nil && match[2] != "" && months[match[1]] != "":
			category, ok := fixtureCategoryIn(t, monthCategories[match[1]], match[2])
			if !ok {
				notFound(w)
				return
			}
			_, _ = io.WriteString(w, `{"data":{"category":`+category+`}}`)
		default:
			notFound(w)
		}
	}))
	t.Cleanup(server.Close)
	h.deps = dependencies{
		baseURL:    server.URL,
		isTerminal: func(io.Writer) bool { return h.terminal },
		lookupEnv:  func(name string) string { return h.env[name] },
		now:        func() time.Time { return fixtureNow },
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "missing"))
	t.Setenv("YNAB_TOKEN", "good-token")
	t.Setenv("YNAB_PLAN", "household")
	t.Setenv("YNAB_ALLOW_WRITES", "")
	if err := os.Unsetenv("YNAB_ALLOW_WRITES"); err != nil {
		t.Fatal(err)
	}
	return h
}

func (h *harness) execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRoot(h.deps)
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs(args)
	err := root.ExecuteContext(t.Context())
	return stdout.String(), err
}

func TestEveryApplicationCommandDeclaresArgsAndGroupsAreRunnable(t *testing.T) {
	var walk func(*cobra.Command)
	walk = func(command *cobra.Command) {
		for _, child := range command.Commands() {
			if child.Name() == "help" || child.Name() == "completion" {
				continue
			}
			if child.Args == nil {
				t.Errorf("%s has no Args validator", child.CommandPath())
			}
			if child.HasSubCommands() && !child.Runnable() {
				t.Errorf("%s has subcommands but no runner", child.CommandPath())
			}
			walk(child)
		}
	}
	walk(NewRoot())
}

func TestExitCodesTopicPrintsSameHelpFromBothEntryPoints(t *testing.T) {
	h := newHarness(t)

	a, err := h.execute(t, "exit-codes")
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.execute(t, "help", "exit-codes")
	if err != nil {
		t.Fatal(err)
	}

	if a != b {
		t.Errorf("exit-codes and help exit-codes differ:\n%s\n%s", a, b)
	}
	for _, row := range []string{"0  Success", "1  Command failure", "2  Usage", "3  Writes disabled"} {
		if !strings.Contains(a, row) {
			t.Errorf("exit-codes topic lacks %q", row)
		}
	}
}

func TestEveryApplicationCommandHasWrappedLongHelp(t *testing.T) {
	var walk func(*cobra.Command)
	walk = func(command *cobra.Command) {
		if command.Name() == "help" || command.Name() == "completion" {
			return
		}
		if command.Long == "" {
			t.Errorf("%s lacks long help", command.CommandPath())
		}
		for _, text := range []string{command.Long, command.Example} {
			for _, line := range strings.Split(text, "\n") {
				if len(line) > 80 {
					t.Errorf("%s line exceeds 80 columns: %q", command.CommandPath(), line)
				}
			}
		}
		for _, child := range command.Commands() {
			walk(child)
		}
	}
	walk(NewRoot())
}

func TestUsageErrorsExitTwoBeforeAnyRequest(t *testing.T) {
	h := newHarness(t)
	cases := [][]string{
		{"unknown"}, {"plans", "extra"}, {"accounts", "list", "extra"}, {"accounts", "get"},
		{"accounts", "get", "a", "b"}, {"config", "extra"}, {"exit-codes", "extra"}, {"output-formats", "extra"},
		{"accounts", "list", "--jsonl", "--csv"}, {"--color", "bold", "plans", "list"}, {"plans", "list", "--nope"},
		{"config", "--config", ""},
	}

	for _, args := range cases {
		out, err := h.execute(t, args...)
		if err == nil || out != "" || ExitCode(err) != ExitUsage {
			t.Errorf("execute(%v) = %q, %v (exit %d), want a usage error", args, out, err, ExitCode(err))
		}
	}
	if len(h.requests) != 0 {
		t.Errorf("usage errors made requests: %v", h.requests)
	}
}

func TestMissingTokenAndPlanExitOneNamingTheSetting(t *testing.T) {
	h := newHarness(t)
	t.Setenv("YNAB_TOKEN", "")
	t.Setenv("YNAB_PLAN", "")

	_, err := h.execute(t, "plans", "list")
	if err == nil || ExitCode(err) != ExitFailure || !strings.Contains(err.Error(), "YNAB_TOKEN") || !strings.Contains(err.Error(), "ynab.toml") {
		t.Errorf("plans list without token = %v (exit %d)", err, ExitCode(err))
	}

	t.Setenv("YNAB_TOKEN", "good-token")
	_, err = h.execute(t, "accounts", "list")
	for _, want := range []string{"plan", "YNAB_PLAN", "--plan", "ynab plans list"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("accounts list without plan = %v, want mention of %q", err, want)
		}
	}
	if ExitCode(err) != ExitFailure || len(h.requests) != 0 {
		t.Errorf("exit %d, requests %v; want exit 1 and no request", ExitCode(err), h.requests)
	}
}

func TestAPIErrorsExitOne(t *testing.T) {
	h := newHarness(t)
	t.Setenv("YNAB_TOKEN", "bad-token")

	out, err := h.execute(t, "plans", "list")

	if err == nil || out != "" || ExitCode(err) != ExitFailure || !strings.Contains(err.Error(), "YNAB_TOKEN") {
		t.Errorf("plans list with bad token = %q, %v (exit %d)", out, err, ExitCode(err))
	}
}

func TestHelpAndVersionNeedNoTokenOrNetwork(t *testing.T) {
	h := newHarness(t)
	t.Setenv("YNAB_TOKEN", "")
	for _, args := range [][]string{{"--help"}, {"--version"}, {"plans"}, {"accounts"}, {"accounts", "get", "--help"}, {"categories"}, {"category-groups"}, {"months"}, {"output-formats"}, {"config", "--help"}} {
		out, err := h.execute(t, args...)
		if err != nil || out == "" {
			t.Errorf("execute(%v) = %q, %v", args, out, err)
		}
	}
	if len(h.requests) != 0 {
		t.Errorf("help made requests: %v", h.requests)
	}
}
