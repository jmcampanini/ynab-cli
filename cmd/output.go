package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/spf13/cobra"
)

// outputFlags selects the record format. Both flags off means the human
// rendering.
type outputFlags struct {
	csv   bool
	jsonl bool
}

// bind adds --jsonl, and --csv when listing, as mutually exclusive flags.
func (f *outputFlags) bind(cmd *cobra.Command, listing bool) {
	cmd.Flags().BoolVar(&f.jsonl, "jsonl", false, "Write one JSON object per line")
	if listing {
		cmd.Flags().BoolVar(&f.csv, "csv", false, "Write CSV with a header row")
		cmd.MarkFlagsMutuallyExclusive("jsonl", "csv")
	}
}

// writeJSONL encodes each record on its own line without HTML escaping.
func writeJSONL[T any](w io.Writer, records []T) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return err
		}
	}
	return nil
}

// writeCSV writes a header of the records' JSON field names and one row
// per record with the same values JSONL would carry: strings unquoted,
// numbers and booleans as JSON text, and absent values empty.
func writeCSV[T any](w io.Writer, records []T) error {
	writer := csv.NewWriter(w)
	if err := writer.Write(jsonFieldNames[T]()); err != nil {
		return err
	}
	for _, record := range records {
		row, err := csvRow(record)
		if err != nil {
			return err
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func jsonFieldNames[T any]() []string {
	var names []string
	for _, field := range reflect.VisibleFields(reflect.TypeFor[T]()) {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name != "" && name != "-" {
			names = append(names, name)
		}
	}
	return names
}

func csvRow(record any) ([]string, error) {
	value := reflect.ValueOf(record)
	var row []string
	for _, field := range reflect.VisibleFields(value.Type()) {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		encoded, err := json.Marshal(value.FieldByIndex(field.Index).Interface())
		if err != nil {
			return nil, err
		}
		text := string(encoded)
		switch {
		case text == "null":
			text = ""
		case strings.HasPrefix(text, `"`):
			if err := json.Unmarshal(encoded, &text); err != nil {
				return nil, err
			}
		}
		row = append(row, text)
	}
	return row, nil
}

// cell is one human table cell: plain text plus an optional paint applied
// after padding so alignment ignores escape codes.
type cell struct {
	paint func(string) string
	text  string
}

func plain(text string) cell { return cell{text: text} }

// column describes a human table column.
type column struct {
	name  string
	right bool
}

// writeTable renders a header and rows as space-separated columns padded
// to the widest cell. The last column is never padded.
func writeTable(w io.Writer, columns []column, rows [][]cell) error {
	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = len(col.name)
	}
	for _, row := range rows {
		for i, c := range row {
			widths[i] = max(widths[i], len(c.text))
		}
	}

	var out strings.Builder
	header := make([]cell, len(columns))
	for i, col := range columns {
		header[i] = plain(col.name)
	}
	for _, row := range append([][]cell{header}, rows...) {
		var line strings.Builder
		for i, c := range row {
			padding := strings.Repeat(" ", widths[i]-len(c.text))
			text := c.text
			if c.paint != nil {
				text = c.paint(text)
			}
			switch {
			case i == len(row)-1 && !columns[i].right:
				line.WriteString(text)
			case columns[i].right:
				line.WriteString(padding + text + "  ")
			default:
				line.WriteString(text + padding + "  ")
			}
		}
		out.WriteString(strings.TrimRight(line.String(), " "))
		out.WriteByte('\n')
	}
	_, err := io.WriteString(w, out.String())
	return err
}

// writeFields renders one record as aligned "name  value" lines.
func writeFields(w io.Writer, fields [][2]string) error {
	width := 0
	for _, field := range fields {
		width = max(width, len(field[0]))
	}
	var out strings.Builder
	for _, field := range fields {
		fmt.Fprintf(&out, "%-*s  %s\n", width, field[0], field[1])
	}
	_, err := io.WriteString(w, out.String())
	return err
}

// countNoun pluralizes a summary count.
func countNoun(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}
