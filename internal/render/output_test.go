package render

import (
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
		t.Errorf("writeTable() =\n%s\nwant\n%s", out.String(), want)
	}
}

func TestWriteCSVFlattensNestedObjectsAndKeepsTimestampsWhole(t *testing.T) {
	type inner struct {
		Code  string `json:"code"`
		Count *int   `json:"count,omitempty"`
	}
	type record struct {
		ID     string    `json:"id"`
		Nested *inner    `json:"nested,omitempty"`
		Seen   time.Time `json:"seen"`
		Skip   string    `json:"-"`
	}
	count := 3
	records := []record{
		{ID: "a", Nested: &inner{Code: "x", Count: &count}, Seen: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Skip: "hidden"},
		{ID: "b"},
	}
	var out strings.Builder

	if err := CSV(&out, records); err != nil {
		t.Fatal(err)
	}

	want := "id,nested_code,nested_count,seen\na,x,3,2026-09-01T00:00:00Z\nb,,,0001-01-01T00:00:00Z\n"
	if out.String() != want {
		t.Errorf("writeCSV() =\n%s\nwant\n%s", out.String(), want)
	}
}

func TestWriteFieldsContinuesMultilineValuesUnderTheValueColumn(t *testing.T) {
	var out strings.Builder

	if err := Fields(&out, [][2]string{{"id", "c1"}, {"note", "line one\n\nline three"}, {"empty", ""}}); err != nil {
		t.Fatal(err)
	}

	want := "id     c1\nnote   line one\n\n       line three\nempty\n"
	if out.String() != want {
		t.Errorf("writeFields() = %q, want %q", out.String(), want)
	}
}
