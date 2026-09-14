package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
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
// numbers and booleans as JSON text, and absent values empty. A nested
// object such as a category's target becomes one column per field, named
// parent_child, all empty when the object is absent.
func writeCSV[T any](w io.Writer, records []T) error {
	fields := jsonFields(reflect.TypeFor[T](), "", nil)
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
			nested, present := fieldValue(value, field.index)
			if !present {
				continue
			}
			text, err := csvCell(nested.Interface())
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

// jsonField is one column CSV output carries: its header name and the
// field index path from the record, through nested objects.
type jsonField struct {
	name  string
	index []int
}

// jsonFields returns the columns of a record type in declaration order,
// so the CSV header and rows use the same columns JSONL carries. Fields
// that JSON encodes as an object of their own fields are flattened with
// the prefix "parent_".
func jsonFields(recordType reflect.Type, prefix string, path []int) []jsonField {
	var fields []jsonField
	for _, field := range reflect.VisibleFields(recordType) {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		index := append(slices.Clone(path), field.Index...)
		if nested := objectType(field.Type); nested != nil {
			fields = append(fields, jsonFields(nested, prefix+name+"_", index)...)
			continue
		}
		fields = append(fields, jsonField{name: prefix + name, index: index})
	}
	return fields
}

var jsonMarshaler = reflect.TypeFor[json.Marshaler]()

// objectType returns the struct type JSON would encode as an object of its
// fields, behind at most one pointer, or nil for scalars and for types
// with their own encoding such as time.Time.
func objectType(fieldType reflect.Type) reflect.Type {
	if fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}
	if fieldType.Kind() != reflect.Struct || fieldType.Implements(jsonMarshaler) || reflect.PointerTo(fieldType).Implements(jsonMarshaler) {
		return nil
	}
	return fieldType
}

// fieldValue walks an index path, dereferencing pointers, and reports
// false when a nil pointer makes the value absent.
func fieldValue(value reflect.Value, index []int) (reflect.Value, bool) {
	for _, i := range index {
		if value.Kind() == reflect.Pointer {
			if value.IsNil() {
				return reflect.Value{}, false
			}
			value = value.Elem()
		}
		value = value.Field(i)
	}
	return value, true
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

// writeFields renders one record as aligned "name  value" lines. A value
// spanning several lines, such as a note, continues under the value
// column.
func writeFields(w io.Writer, fields [][2]string) error {
	width := 0
	for _, field := range fields {
		width = max(width, utf8.RuneCountInString(field[0]))
	}
	continuation := "\n" + strings.Repeat(" ", width+2)
	var out strings.Builder
	for _, field := range fields {
		line := fmt.Sprintf("%-*s  %s", width, field[0], strings.ReplaceAll(field[1], "\n", continuation))
		for _, part := range strings.Split(line, "\n") {
			out.WriteString(strings.TrimRight(part, " "))
			out.WriteByte('\n')
		}
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
