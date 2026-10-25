package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jmcampanini/ynab-cli/internal/categoryview"
	"github.com/jmcampanini/ynab-cli/internal/ynab"
	"github.com/spf13/cobra"
)

func TestFutureRefillWarningConditions(t *testing.T) {
	refill, whole := false, true
	zero, short := ynab.Amount(0), ynab.Amount(12500)
	cases := []struct {
		name       string
		month      string
		target     *categoryview.Target
		internal   bool
		wantAmount string
	}{
		{"zero underfunding", "2026-10", &categoryview.Target{Type: "NEED", NeedsWholeAmount: &refill, Underfunded: &zero}, false, "0.00"},
		{"positive underfunding", "2026-10", &categoryview.Target{Type: "NEED", NeedsWholeAmount: &refill, Underfunded: &short}, false, "12.50"},
		{"missing underfunding", "2026-10", &categoryview.Target{Type: "NEED", NeedsWholeAmount: &refill}, false, "not reported"},
		{"current UTC month", "2026-09", &categoryview.Target{Type: "NEED", NeedsWholeAmount: &refill}, false, ""},
		{"past month", "2026-08", &categoryview.Target{Type: "NEED", NeedsWholeAmount: &refill}, false, ""},
		{"set aside", "2026-10", &categoryview.Target{Type: "NEED", NeedsWholeAmount: &whole}, false, ""},
		{"unknown rollover", "2026-10", &categoryview.Target{Type: "NEED"}, false, ""},
		{"other target", "2026-10", &categoryview.Target{Type: "MF", NeedsWholeAmount: &refill}, false, ""},
		{"no target", "2026-10", nil, false, ""},
		{"internal", "2026-10", &categoryview.Target{Type: "NEED", NeedsWholeAmount: &refill}, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := &app{deps: dependencies{now: func() time.Time {
				return time.Date(2026, time.August, 31, 22, 0, 0, 0, time.FixedZone("UTC-4", -4*60*60))
			}}}
			var stderr bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetErr(&stderr)
			record := categoryview.Record{ID: "c1", Name: "Internet", Group: "Bills", Month: tc.month, Target: tc.target, Internal: tc.internal}

			a.warnFutureRefill(cmd, []categoryview.Record{record}, nil)

			if tc.wantAmount == "" {
				if stderr.Len() != 0 {
					t.Errorf("unexpected warning: %s", stderr.String())
				}
				return
			}
			for _, part := range []string{"warning [future-refill-funding]", "Bills: Internet (c1)", tc.month, "API underfunded " + tc.wantAmount, "may still require funding", "Check the web target panel"} {
				if !strings.Contains(stderr.String(), part) {
					t.Errorf("warning %q lacks %q", stderr.String(), part)
				}
			}
		})
	}
}

func newFutureRefillHarness(t *testing.T) *harness {
	t.Helper()
	h := newHarness(t)
	h.deps.now = func() time.Time { return time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC) }
	h.categoryGroups = strings.Replace(fixtureCategoryGroups, `"goal_needs_whole_amount":true`, `"goal_needs_whole_amount":false`, 1)
	h.categoryGroups = strings.Replace(h.categoryGroups, `"goal_type":"TBD","goal_needs_whole_amount":null`, `"goal_type":"NEED","goal_needs_whole_amount":false`, 1)
	return h
}

func TestFutureRefillWarningsReachCommandStderr(t *testing.T) {
	cases := []struct {
		name      string
		args      []string
		wantEmpty bool
	}{
		{"fund named once", []string{"months", "fund", "c1", "Bills: Internet", "--allow-writes"}, false},
		{"fund empty JSONL dry run", []string{"months", "fund", "--all-underfunded", "--dry-run", "--jsonl"}, true},
		{"fund empty CSV", []string{"months", "fund", "--all-underfunded", "--allow-writes", "--csv"}, true},
		{"filtered funding JSONL", []string{"reports", "funding", "--underfunded", "--jsonl"}, true},
		{"filtered funding CSV", []string{"reports", "funding", "--underfunded", "--csv"}, true},
		{"funding table", []string{"reports", "funding"}, false},
		{"category list JSONL", []string{"categories", "list", "--jsonl"}, false},
		{"category get table", []string{"categories", "get", "c1"}, false},
		{"month JSONL", []string{"months", "get", "2026-09", "--jsonl"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newFutureRefillHarness(t)
			args := append([]string(nil), tc.args...)
			if tc.args[0] != "months" || tc.args[1] != "get" {
				args = append(args, "--month", "2026-09")
			}

			stdout, stderr, err := h.executeStreams(t, args...)

			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(stderr, "warning [future-refill-funding]") != 1 || !strings.Contains(stderr, "Bills: Internet (c1), 2026-09: API underfunded $0.00") {
				t.Errorf("stderr = %q, want one visible-category warning", stderr)
			}
			if strings.Contains(stdout, "future-refill-funding") {
				t.Errorf("warning leaked into stdout: %q", stdout)
			}
			if tc.wantEmpty {
				if strings.Contains(strings.Join(args, " "), "--csv") {
					rows, err := csv.NewReader(strings.NewReader(stdout)).ReadAll()
					if err != nil || len(rows) != 1 || rows[0][0] != "id" {
						t.Errorf("CSV = %q, error %v; want only a header", stdout, err)
					}
				} else if stdout != "" {
					t.Errorf("stdout = %q, want no records", stdout)
				}
			}
			if len(h.writes) != 0 {
				t.Errorf("zero underfunding produced writes: %v", h.writes)
			}
			if len(h.requests) != 3 {
				t.Errorf("requests = %v, want only the existing three reads", h.requests)
			}
			if strings.Contains(strings.Join(args, " "), "--jsonl") && stdout != "" {
				for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
					if !json.Valid([]byte(line)) {
						t.Errorf("invalid JSONL record: %q", line)
					}
				}
			}
		})
	}
}

func TestFutureRefillWarningPreservesFundingAmount(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "write", true: "dry run"}[dryRun], func(t *testing.T) {
			h := newFutureRefillHarness(t)
			h.categoryGroups = strings.Replace(h.categoryGroups, `"goal_under_funded":0`, `"goal_under_funded":12000`, 1)
			args := []string{"months", "fund", "c1", "--month", "2026-09", "--allow-writes", "--force", "--jsonl"}
			if dryRun {
				args = append(args, "--dry-run")
			}

			stdout, stderr, err := h.executeStreams(t, args...)

			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stderr, "API underfunded $12.00") {
				t.Errorf("warning = %q, want original API amount", stderr)
			}
			var record struct {
				Assigned float64 `json:"assigned"`
			}
			if err := json.Unmarshal([]byte(stdout), &record); err != nil {
				t.Fatal(err)
			}
			if record.Assigned != 92 {
				t.Errorf("assigned = %v, want 92", record.Assigned)
			}
			var wantWrites []string
			if !dryRun {
				wantWrites = []string{`PATCH /plans/p1/months/2026-09-01/categories/c1 {"category":{"budgeted":92000}}`}
			}
			if !reflect.DeepEqual(h.writes, wantWrites) {
				t.Errorf("writes = %v, want %v", h.writes, wantWrites)
			}
		})
	}
}

func TestFutureRefillWarningsIncludeExplicitHiddenCategories(t *testing.T) {
	for _, args := range [][]string{
		{"categories", "list", "--hidden"},
		{"categories", "get", "c5"},
		{"months", "fund", "c5", "--dry-run", "--force"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			h := newFutureRefillHarness(t)

			_, stderr, err := h.executeStreams(t, append(args, "--month", "2026-09")...)

			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stderr, "Wishes: Internet (c5)") {
				t.Errorf("stderr = %q, want warning for explicitly included hidden category", stderr)
			}
		})
	}
}
