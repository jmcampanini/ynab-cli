package render

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func TestWriteTablePadsByRuneCount(t *testing.T) {
	var out strings.Builder
	columns := []Column{{Name: "NAME"}, {Name: "BALANCE", Right: true}, {Name: "NOTE"}}
	rows := [][]Cell{
		{Plain("Épargne"), Plain("-1.234,56 €"), Plain("x")},
		{Plain("Cash"), Plain("5,00 €"), Plain("y")},
	}

	if err := Table(&out, columns, rows); err != nil {
		t.Fatal(err)
	}

	want := "NAME         BALANCE  NOTE\nÉpargne  -1.234,56 €  x\nCash          5,00 €  y\n"
	if out.String() != want {
		t.Errorf("Table() =\n%s\nwant\n%s", out.String(), want)
	}
}

func TestWriteCSVKeepsNestedValuesInOneCell(t *testing.T) {
	type inner struct {
		Code  string `json:"code"`
		Count *int   `json:"count,omitempty"`
	}
	type record struct {
		ID     string    `json:"id"`
		Nested *inner    `json:"nested,omitempty"`
		Seen   time.Time `json:"seen"`
		Items  []string  `json:"items,omitempty"`
		Skip   string    `json:"-"`
	}
	count := 3
	records := []record{
		{ID: "a", Nested: &inner{Code: "x", Count: &count}, Seen: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Items: []string{"one", "two"}, Skip: "hidden"},
		{ID: "b"},
	}
	var out strings.Builder

	if err := CSV(&out, records); err != nil {
		t.Fatal(err)
	}

	want := "id,nested,seen,items\na,\"{\"\"code\"\":\"\"x\"\",\"\"count\"\":3}\",2026-09-01T00:00:00Z,\"[\"\"one\"\",\"\"two\"\"]\"\nb,,0001-01-01T00:00:00Z,\n"
	if out.String() != want {
		t.Errorf("CSV() =\n%s\nwant\n%s", out.String(), want)
	}
}

func TestWriteCSVWithoutHeaderAndWithEmbeddedRecord(t *testing.T) {
	type record struct {
		Name string `json:"name"`
	}
	type marked struct {
		record
		DryRun bool `json:"dry_run,omitempty"`
	}
	var out strings.Builder
	if err := CSV(&out, []marked{{record: record{Name: "a,b"}, DryRun: true}}, true); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "\"a,b\",true\n"; got != want {
		t.Fatalf("CSV() = %q, want %q", got, want)
	}
	out.Reset()
	if err := CSV(&out, []marked{}, true); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("empty CSV without a header = %q", out.String())
	}
	if err := CSV(&out, []marked{}); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "name,dry_run\n"; got != want {
		t.Fatalf("empty CSV() = %q, want %q", got, want)
	}
}

func TestCSVReportsEncodingAndFlushFailures(t *testing.T) {
	type record struct {
		Value float64 `json:"value"`
	}
	var out strings.Builder
	if err := CSV(&out, []record{{Value: math.NaN()}}); err == nil {
		t.Fatal("CSV accepted a value JSON cannot encode")
	}
	wantErr := errors.New("write failed")
	if err := CSV(failingWriter{err: wantErr}, []record{{Value: 1}}); !errors.Is(err, wantErr) {
		t.Fatalf("CSV flush error = %v, want %v", err, wantErr)
	}
}

func TestJSONCollectionAndEmptyRecords(t *testing.T) {
	type record struct {
		Name string `json:"name"`
	}
	for _, records := range [][]record{nil, {}, {{Name: "<groceries> & food"}}} {
		var out strings.Builder
		if err := JSON(&out, records, true); err != nil {
			t.Fatal(err)
		}
		var result struct {
			Records    []record `json:"records"`
			HasMore    bool     `json:"hasMore"`
			NextCursor *string  `json:"nextCursor"`
			Complete   bool     `json:"complete"`
		}
		if err := json.Unmarshal([]byte(out.String()), &result); err != nil {
			t.Fatal(err)
		}
		if result.Records == nil || len(result.Records) != len(records) || !result.HasMore || result.NextCursor != nil || !result.Complete {
			t.Fatalf("JSON collection = %s", out.String())
		}
		if strings.Contains(out.String(), `\u003c`) || !strings.HasSuffix(out.String(), "\n") {
			t.Fatalf("JSON formatting = %q", out.String())
		}
	}
}

func TestCSVPreservesQuotesAndMultilineStrings(t *testing.T) {
	type record struct {
		Name string `json:"name"`
	}
	want := "a, \"quoted\" name\nnext line"
	var out strings.Builder
	if err := CSV(&out, []record{{Name: want}}); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(out.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || len(rows[1]) != 1 || rows[1][0] != want {
		t.Fatalf("CSV round trip = %#v", rows)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWriteFieldsContinuesMultilineValuesUnderTheValueColumn(t *testing.T) {
	var out strings.Builder

	if err := Fields(&out, [][2]string{{"id", "c1"}, {"note", "line one\n\nline three"}, {"empty", ""}}); err != nil {
		t.Fatal(err)
	}

	want := "id     c1\nnote   line one\n\n       line three\nempty\n"
	if out.String() != want {
		t.Errorf("Fields() = %q, want %q", out.String(), want)
	}
}
