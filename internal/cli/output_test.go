package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmcampanini/ynab-cli/internal/config"
	"github.com/jmcampanini/ynab-cli/internal/status"
)

type collectionResult struct {
	Records    []json.RawMessage `json:"records"`
	HasMore    bool              `json:"hasMore"`
	NextCursor *string           `json:"nextCursor"`
	Complete   bool              `json:"complete"`
}

func decodeCollection(t *testing.T, text string) collectionResult {
	t.Helper()
	var result collectionResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatal(err)
	}
	if result.Records == nil || !result.Complete || result.NextCursor != nil {
		t.Fatalf("invalid collection status: %s", text)
	}
	return result
}

func TestReadCommandsWriteJSONCollectionsAndCSV(t *testing.T) {
	commands := [][]string{
		{"plans", "list"}, {"plans", "get"}, {"plans", "status"},
		{"accounts", "list"}, {"accounts", "get", "a1"},
		{"categories", "list"}, {"categories", "get", "c1"},
		{"category-groups", "list"}, {"months", "list"}, {"months", "get"},
		{"transactions", "list"}, {"transactions", "get", "t1"},
		{"transactions", "review"}, {"payees", "list"}, {"payees", "get", "p2"},
		{"scheduled", "list"}, {"scheduled", "get", "st2"},
		{"money-movements", "list"}, {"reports", "funding"}, {"reports", "spending"},
	}
	for _, command := range commands {
		t.Run(strings.Join(command, " "), func(t *testing.T) {
			h := newHarness(t)
			text, err := h.execute(t, append(command, "--json", "--color", "always")...)
			if err != nil {
				t.Fatal(err)
			}
			result := decodeCollection(t, text)
			if len(result.Records) == 0 || result.HasMore || strings.Contains(text, "\x1b") {
				t.Fatalf("unexpected JSON collection: %s", text)
			}
			text, err = h.execute(t, append(command, "--csv", "--no-header", "--color", "always")...)
			if err != nil {
				t.Fatal(err)
			}
			rows, err := csv.NewReader(strings.NewReader(text)).ReadAll()
			if err != nil || len(rows) != len(result.Records) || strings.Contains(text, "\x1b") {
				t.Fatalf("CSV rows = %d, want %d; error %v; output %s", len(rows), len(result.Records), err, text)
			}
		})
	}
}

func TestReadListsCapRecordsAndReportMore(t *testing.T) {
	commands := [][]string{
		{"plans", "list"}, {"accounts", "list"}, {"categories", "list"},
		{"category-groups", "list"}, {"months", "list"}, {"transactions", "list"},
		{"transactions", "review"}, {"payees", "list"}, {"scheduled", "list"},
		{"money-movements", "list"}, {"reports", "funding"}, {"reports", "spending"},
	}
	for _, command := range commands {
		t.Run(strings.Join(command, " "), func(t *testing.T) {
			h := newHarness(t)
			text, err := h.execute(t, append(command, "--json", "--limit", "1")...)
			if err != nil {
				t.Fatal(err)
			}
			result := decodeCollection(t, text)
			if len(result.Records) != 1 || !result.HasMore {
				t.Fatalf("capped collection = %s", text)
			}
			for _, format := range []string{"--jsonl", "--csv"} {
				text, diagnostics, err := h.executeStreams(t, append(command, format, "--limit", "1")...)
				if err != nil || !strings.Contains(diagnostics, "hasMore=true, nextCursor=null") {
					t.Fatalf("error %v; diagnostics %q", err, diagnostics)
				}
				wantLines := 1
				if format == "--csv" {
					wantLines++
				}
				if strings.Count(text, "\n") != wantLines {
					t.Fatalf("%s capped output = %s", format, text)
				}
			}
		})
	}
}

func TestDefaultRecordLimitAndAll(t *testing.T) {
	h := newHarness(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"data":{"plans":[`)
		for i := range config.DefaultRecordLimit + 1 {
			if i > 0 {
				_, _ = io.WriteString(w, ",")
			}
			_, _ = fmt.Fprintf(w, `{"id":"p%d","name":"Plan %d"}`, i, i)
		}
		_, _ = io.WriteString(w, `]}}`)
	}))
	defer server.Close()
	h.deps.baseURL = server.URL
	for _, tc := range []struct {
		flags []string
		count int
		more  bool
	}{
		{nil, config.DefaultRecordLimit, true},
		{[]string{"--all"}, config.DefaultRecordLimit + 1, false},
		{[]string{"--limit", "0"}, 0, true},
	} {
		text, err := h.execute(t, append([]string{"plans", "list", "--json"}, tc.flags...)...)
		if err != nil {
			t.Fatal(err)
		}
		result := decodeCollection(t, text)
		if len(result.Records) != tc.count || result.HasMore != tc.more {
			t.Errorf("flags %v: got %d records, hasMore=%t", tc.flags, len(result.Records), result.HasMore)
		}
	}
}

func TestOutputValidationPrecedesWork(t *testing.T) {
	for _, flags := range [][]string{
		{"--json", "--csv"}, {"--json", "--jsonl"}, {"--csv", "--jsonl"},
		{"--limit", "1", "--all"}, {"--limit", "-1"}, {"--limit", "bad"},
		{"--no-header"}, {"--overwrite"}, {"--output-file", ""},
		{"--output-file", "unused.json"}, {"--cursor", "anything"},
	} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			h := newHarness(t)
			out, err := h.execute(t, append([]string{"plans", "list"}, flags...)...)
			if err == nil || out != "" || len(h.requests) != 0 {
				t.Fatalf("out %q, error %v, requests %v", out, err, h.requests)
			}
		})
	}
}

func TestCategoryLimitKeepsOnlyShownRowsInGroupSubtotal(t *testing.T) {
	h := newHarness(t)
	out, diagnostics, err := h.executeStreams(t, "categories", "list", "--limit", "1")
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Bills", "Internet", "$80.00", "1 category, 2 hidden not shown\n"} {
		if !strings.Contains(out, text) {
			t.Errorf("capped categories lack %q: %s", text, out)
		}
	}
	for _, text := range []string{"Rent", "Dining Out", "$1,580.00"} {
		if strings.Contains(out, text) {
			t.Errorf("capped categories include omitted row or subtotal %q: %s", text, out)
		}
	}
	if !strings.Contains(diagnostics, "hasMore=true") {
		t.Errorf("missing truncation diagnostic: %s", diagnostics)
	}
}

func TestCollectionLimitPreservesFullSummaryTotals(t *testing.T) {
	for _, command := range [][]string{
		{"reports", "funding"}, {"reports", "spending"},
		{"transactions", "list"}, {"transactions", "review"},
	} {
		t.Run(strings.Join(command, " "), func(t *testing.T) {
			h := newHarness(t)
			full, err := h.execute(t, command...)
			if err != nil {
				t.Fatal(err)
			}
			limited, diagnostics, err := h.executeStreams(t, append(command, "--limit", "1")...)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSuffix(full, "\n"), "\n")
			if !strings.HasSuffix(limited, lines[len(lines)-1]+"\n") || !strings.Contains(diagnostics, "hasMore=true") {
				t.Errorf("limited summary differs from full collection: %s\nfull: %s\ndiagnostics: %s", limited, full, diagnostics)
			}
		})
	}
}

func TestEmptyCollectionFormats(t *testing.T) {
	h := newHarness(t)
	for _, tc := range []struct {
		flags []string
		want  string
	}{
		{[]string{"--json"}, "{\"records\":[],\"hasMore\":false,\"nextCursor\":null,\"complete\":true}\n"},
		{[]string{"--jsonl"}, ""},
		{[]string{"--csv", "--no-header"}, ""},
	} {
		args := append([]string{"transactions", "list", "--memo", "no fixture has this memo"}, tc.flags...)
		out, diagnostics, err := h.executeStreams(t, args...)
		if err != nil || out != tc.want || diagnostics != "" {
			t.Errorf("%v: output %q, diagnostics %q, error %v", tc.flags, out, diagnostics, err)
		}
	}
}

func TestOutputFileReplacesOnlyWhenRequested(t *testing.T) {
	h := newHarness(t)
	path := filepath.Join(t.TempDir(), "plans.jsonl")
	if err := os.WriteFile(path, []byte("previous\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := h.execute(t, "plans", "list", "--jsonl", "--output-file", path)
	if status.Code(err) != status.ExitFailure || out != "" || len(h.requests) != 0 {
		t.Fatalf("existing destination: out %q, error %v, requests %v", out, err, h.requests)
	}
	out, err = h.execute(t, "plans", "list", "--jsonl", "--output-file", path, "--overwrite")
	if err != nil || out != "" {
		t.Fatalf("export: out %q, error %v", out, err)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := h.execute(t, "plans", "list", "--jsonl")
	if err != nil || string(bytes) != want {
		t.Fatalf("file %q, want %q; error %v", bytes, want, err)
	}
}

func TestFailedReadWritesNoRecordsAndPreservesOutputFile(t *testing.T) {
	h := newHarness(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/plans" {
			_, _ = io.WriteString(w, fixturePlans)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"accounts":[{"id":"a1"},`)
	}))
	defer server.Close()
	h.deps.baseURL = server.URL
	for _, format := range []string{"--json", "--jsonl", "--csv"} {
		out, err := h.execute(t, "accounts", "list", format)
		if err == nil || out != "" {
			t.Fatalf("%s failed read: out %q, error %v", format, out, err)
		}
		path := filepath.Join(t.TempDir(), "export")
		if err := os.WriteFile(path, []byte("previous\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		out, err = h.execute(t, "accounts", "list", format, "--output-file", path, "--overwrite")
		if err == nil || out != "" {
			t.Fatalf("%s failed export: out %q, error %v", format, out, err)
		}
		bytes, err := os.ReadFile(path)
		if err != nil || string(bytes) != "previous\n" {
			t.Fatalf("failed export changed destination to %q; error %v", bytes, err)
		}
	}
}
