package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/jmcampanini/ynab-cli/internal/status"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
)

const deleteSplitLines = `[
{"id":"s1","amount":-2500,"category_id":"c1","deleted":false},
{"id":"s2","amount":-1500,"category_id":"c3","deleted":false}]`

type splitDeleteScenario struct {
	accountID string
	failure   string
	lines     string
	transfer  string
}

type splitDeleteRequests struct {
	deleted   bool
	reads     int
	repairs   []string
	temporary map[string]bool
}

func TestSplitDeleteRecalculatesEachDistinctCategoryWithoutReadingTotals(t *testing.T) {
	tests := []struct {
		name       string
		scenario   splitDeleteScenario
		wantRepair []string
	}{
		{name: "categorized split", wantRepair: []string{"c1", "c3"}},
		{
			name: "repeated category and deleted lines",
			scenario: splitDeleteScenario{lines: `[
{"id":"s1","amount":-1000,"category_id":"c1"},
{"id":"s2","amount":-1500,"category_id":"c1"},
{"id":"s3","amount":-1500,"category_id":"c3"},
{"id":"old","amount":-9000,"category_id":"other","deleted":true}]`},
			wantRepair: []string{"c1", "c3"},
		},
		{
			name:       "zero net contribution still receives the workaround",
			scenario:   splitDeleteScenario{lines: `[{"amount":2500,"category_id":"c1"},{"amount":-2500,"category_id":"c1"},{"amount":-4000,"category_id":"c3"}]`},
			wantRepair: []string{"c1", "c3"},
		},
		{
			name: "transfer and uncategorized lines are excluded",
			scenario: splitDeleteScenario{lines: `[
{"amount":-2500,"category_id":"c1"},
{"amount":-500,"category_id":"c3","transfer_account_id":"a2"},
{"amount":-1000,"category_id":null}]`},
			wantRepair: []string{"c1"},
		},
		{name: "ordinary transaction", scenario: splitDeleteScenario{lines: "[]"}},
		{name: "tracking account", scenario: splitDeleteScenario{accountID: "a3"}},
		{name: "transfer parent", scenario: splitDeleteScenario{transfer: "a2"}},
		{name: "uncategorized split", scenario: splitDeleteScenario{lines: `[{"amount":-1000},{"amount":-3000}]`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, seen := newSplitDeleteHarness(t, tt.scenario)

			out, stderr, err := h.executeStreams(t, "transactions", "delete", "split", "--allow-writes", "--jsonl")

			if err != nil {
				t.Fatal(err)
			}
			assertDeletedSplitOutput(t, out)
			if !seen.deleted || !reflect.DeepEqual(seen.repairs, tt.wantRepair) || len(seen.temporary) != 0 {
				t.Errorf("delete state = %+v, want original deleted, repairs %v and no temporary entries", seen, tt.wantRepair)
			}
			if got := strings.Count(stderr, "warning [split-delete-recalculated]"); got != len(tt.wantRepair) {
				t.Errorf("diagnostics = %q, want %d completed create/delete pairs", stderr, len(tt.wantRepair))
			}
			if seen.reads != 3 {
				t.Errorf("reads = %d, want only plans, accounts and transaction", seen.reads)
			}
		})
	}
}

func TestSplitDeleteFailuresPreserveTheDeletedRecord(t *testing.T) {
	tests := []struct {
		name            string
		failure         string
		wantDeleted     bool
		wantError       string
		wantRepair      []string
		wantTemporary   int
		wantDiagnostics int
	}{
		{name: "delete fails", failure: "delete", wantError: "internal_server_error"},
		{name: "create fails", failure: "create", wantDeleted: true, wantError: "inspect the register for memo"},
		{name: "create returns no ID", failure: "create-id", wantDeleted: true, wantError: "returned no ID"},
		{name: "temporary deletion fails", failure: "temporary-delete", wantDeleted: true, wantError: "temporary zero-amount transaction temporary-c1", wantRepair: []string{"c1"}, wantTemporary: 1},
		{name: "later category fails", failure: "second-create", wantDeleted: true, wantError: "inspect the register for memo", wantRepair: []string{"c1"}, wantDiagnostics: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, seen := newSplitDeleteHarness(t, splitDeleteScenario{failure: tt.failure})

			out, stderr, err := h.executeStreams(t, "transactions", "delete", "split", "--allow-writes", "--jsonl")

			if err == nil || status.Code(err) != status.ExitFailure || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("delete error = %v, want exit 1 containing %q", err, tt.wantError)
			}
			if tt.wantDeleted {
				assertDeletedSplitOutput(t, out)
				if !strings.Contains(err.Error(), "transaction split was deleted") || !strings.Contains(err.Error(), "later categories were not processed") {
					t.Errorf("error does not explain completed deletion and incomplete repair: %v", err)
				}
			} else if out != "" {
				t.Errorf("output before deletion = %q, want none", out)
			}
			if seen.deleted != tt.wantDeleted || !reflect.DeepEqual(seen.repairs, tt.wantRepair) || len(seen.temporary) != tt.wantTemporary {
				t.Errorf("delete state = %+v, want deleted %t, repairs %v, remaining %d", seen, tt.wantDeleted, tt.wantRepair, tt.wantTemporary)
			}
			if got := strings.Count(stderr, "warning [split-delete-recalculated]"); got != tt.wantDiagnostics {
				t.Errorf("diagnostics = %q, want %d completed create/delete pairs", stderr, tt.wantDiagnostics)
			}
		})
	}
}

func TestSplitDeleteDryRunDoesNotCreateTemporaryEntries(t *testing.T) {
	h, seen := newSplitDeleteHarness(t, splitDeleteScenario{})

	out, stderr, err := h.executeStreams(t, "transactions", "delete", "split", "--dry-run", "--jsonl")

	if err != nil || stderr != "" || !strings.Contains(out, `"dry_run":true`) {
		t.Fatalf("dry run = %q, %q, %v", out, stderr, err)
	}
	if seen.deleted || seen.reads != 3 || len(seen.repairs) != 0 {
		t.Errorf("dry run state = %+v, want three reads without deletion or repairs", seen)
	}
}

func TestRawSplitDeleteDoesNotApplyTheWorkaround(t *testing.T) {
	h, seen := newSplitDeleteHarness(t, splitDeleteScenario{})

	_, stderr, err := h.executeStreams(t, "api", "delete", "/plans/p1/transactions/split", "--allow-writes")

	if err != nil || stderr != "" {
		t.Fatalf("raw delete = %q, %v", stderr, err)
	}
	if !seen.deleted || seen.reads != 0 || len(seen.repairs) != 0 {
		t.Errorf("raw delete state = %+v, want only the requested DELETE", seen)
	}
}

func TestSplitDeleteRemovesTemporaryEntryAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var removed bool
	s := writeSession{client: &ynab.Client{HTTPClient: &http.Client{Transport: splitRepairTransport(func(r *http.Request) (*http.Response, error) {
		body := `{"data":{"transaction":{"id":"temporary"}}}`
		responseBody := io.NopCloser(strings.NewReader(body))
		switch r.Method {
		case http.MethodPost:
			responseBody = &cancelOnClose{ReadCloser: responseBody, cancel: cancel}
		case http.MethodDelete:
			if ctx.Err() == nil || r.Context().Err() != nil || r.URL.Path != "/v1/plans/p1/transactions/temporary" {
				t.Errorf("cleanup request = %s, original context %v, cleanup context %v", r.URL.Path, ctx.Err(), r.Context().Err())
			}
			if _, ok := r.Context().Deadline(); !ok {
				t.Error("cleanup context has no deadline")
			}
			removed = true
		default:
			t.Errorf("unexpected method: %s", r.Method)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: responseBody}, nil
	})}}, plan: ynab.Plan{ID: "p1"}}

	err := s.recalculateSplitCategory(ctx, ynab.Transaction{ID: "split", AccountID: "a1", Date: "2026-08-12"}, ynab.Category{ID: "c1"})

	if err != nil || !removed {
		t.Errorf("cleanup after cancellation = %v, removed %t; want successful removal", err, removed)
	}
}

type splitRepairTransport func(*http.Request) (*http.Response, error)

func (f splitRepairTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *cancelOnClose) Close() error {
	r.cancel()
	return r.ReadCloser.Close()
}

func assertDeletedSplitOutput(t *testing.T, out string) {
	t.Helper()
	var record struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &record); err != nil || record.ID != "split" || strings.Count(out, "\n") != 1 {
		t.Errorf("output = %q, %v; want only the original deleted record", out, err)
	}
}

func newSplitDeleteHarness(t *testing.T, scenario splitDeleteScenario) (*harness, *splitDeleteRequests) {
	t.Helper()
	h := newHarness(t)
	seen := &splitDeleteRequests{temporary: map[string]bool{}}
	if scenario.lines == "" {
		scenario.lines = deleteSplitLines
	}
	if scenario.accountID == "" {
		scenario.accountID = "a1"
	}
	transfer := "null"
	if scenario.transfer != "" {
		transfer = fmt.Sprintf("%q", scenario.transfer)
	}
	transaction := fmt.Sprintf(`{"id":"split","date":"2026-08-12","account_id":%q,"account_name":"Checking","amount":-4000,"cleared":"uncleared","deleted":false,"transfer_account_id":%s,"subtransactions":%s}`, scenario.accountID, transfer, scenario.lines)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			seen.reads++
		}
		stage := ""
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/plans":
			_, _ = io.WriteString(w, fixturePlans)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/plans/p1/accounts":
			_, _ = io.WriteString(w, fixtureAccounts)
			return
		case r.Method == http.MethodGet && r.URL.Path == "/plans/p1/transactions/split":
			_, _ = fmt.Fprintf(w, `{"data":{"transaction":%s}}`, transaction)
			return
		case r.Method == http.MethodDelete && r.URL.Path == "/plans/p1/transactions/split":
			stage = "delete"
		case r.Method == http.MethodPost && r.URL.Path == "/plans/p1/transactions":
			stage = "create"
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/plans/p1/transactions/temporary-"):
			stage = "temporary-delete"
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
			notFound(w)
			return
		}
		if scenario.failure == stage || (scenario.failure == "second-create" && stage == "create" && len(seen.repairs) == 1) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error":{"id":"500","name":"internal_server_error","detail":"fixture failure"}}`)
			return
		}
		switch stage {
		case "delete":
			seen.deleted = true
			_, _ = fmt.Fprintf(w, `{"data":{"transaction":%s}}`, strings.ReplaceAll(transaction, `"deleted":false`, `"deleted":true`))
		case "create":
			if !seen.deleted {
				t.Error("temporary entry created before original deletion")
			}
			var body struct {
				Transaction map[string]any `json:"transaction"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			category, _ := body.Transaction["category_id"].(string)
			want := map[string]any{
				"account_id": scenario.accountID, "approved": false, "category_id": category, "cleared": "uncleared",
				"date": "2026-08-12", "memo": "ynab-cli: recalculate category after deleting split split", "amount": float64(0),
			}
			if !reflect.DeepEqual(body.Transaction, want) || (category != "c1" && category != "c3") {
				t.Errorf("temporary entry = %v, want %v for an affected category only", body.Transaction, want)
			}
			if scenario.failure == "create-id" {
				_, _ = io.WriteString(w, `{"data":{"transaction":{}}}`)
				return
			}
			seen.repairs = append(seen.repairs, category)
			id := "temporary-" + category
			seen.temporary[id] = true
			_, _ = fmt.Fprintf(w, `{"data":{"transaction":{"id":%q}}}`, id)
		case "temporary-delete":
			id := strings.TrimPrefix(r.URL.Path, "/plans/p1/transactions/")
			if !seen.temporary[id] {
				t.Errorf("delete of unknown temporary entry %q", id)
			}
			delete(seen.temporary, id)
			_, _ = fmt.Fprintf(w, `{"data":{"transaction":{"id":%q,"deleted":true}}}`, id)
		}
	}))
	t.Cleanup(server.Close)
	h.deps.baseURL = server.URL
	return h, seen
}
