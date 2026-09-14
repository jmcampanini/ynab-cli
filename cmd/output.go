package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"

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
	fields := jsonFields(reflect.TypeFor[T]())
	header := make([]string, len(fields))
	for i, field := range fields {
		header[i] = field.name
	}

	writer := csv.NewWriter(w)
	if err := writer.Write(header); err != nil {
		return err
	}
	for _, record := range records {
		value := reflect.ValueOf(record)
		row := make([]string, len(fields))
		for i, field := range fields {
			text, err := csvCell(value.FieldByIndex(field.index).Interface())
			if err != nil {
				return err
			}
			row[i] = text
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

// jsonField is one struct field that JSON output carries: its JSON name and
// its index for reflect.Value.FieldByIndex.
type jsonField struct {
	name  string
	index []int
}

// jsonFields returns the fields of a record type that JSON output carries,
// in declaration order, so the CSV header and rows use the same columns.
func jsonFields(recordType reflect.Type) []jsonField {
	var fields []jsonField
	for _, field := range reflect.VisibleFields(recordType) {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name != "" && name != "-" {
			fields = append(fields, jsonField{name: name, index: field.Index})
		}
	}
	return fields
}

// csvCell renders one field value as its JSON text, with strings unquoted
// and null empty.
func csvCell(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	text := string(encoded)
	switch {
	case text == "null":
		return "", nil
	case strings.HasPrefix(text, `"`):
		if err := json.Unmarshal(encoded, &text); err != nil {
			return "", err
		}
	}
	return text, nil
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
		widths[i] = utf8.RuneCountInString(col.name)
	}
	for _, row := range rows {
		for i, c := range row {
			widths[i] = max(widths[i], utf8.RuneCountInString(c.text))
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
			padding := strings.Repeat(" ", widths[i]-utf8.RuneCountInString(c.text))
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
		width = max(width, utf8.RuneCountInString(field[0]))
	}
	var out strings.Builder
	for _, field := range fields {
		out.WriteString(strings.TrimRight(fmt.Sprintf("%-*s  %s", width, field[0], field[1]), " "))
		out.WriteByte('\n')
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
