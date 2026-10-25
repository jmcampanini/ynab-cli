// Package render encodes machine output and formats human output without terminal I/O policy.
package render

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// JSONL encodes each record on its own line without HTML escaping.
func JSONL[T any](w io.Writer, records []T) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return err
		}
	}
	return nil
}

// CSV writes a header of the records' JSON field names and one row
// per record with the same values JSONL would carry: strings unquoted,
// numbers and booleans as JSON text, and absent values empty. A nested
// object such as a category's target becomes one column per field, named
// parent_child, all empty when the object is absent. Array fields are
// left out; callers flatten them to rows first.
func CSV[T any](w io.Writer, records []T) error {
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
// the prefix "parent_"; fields that encode as arrays are skipped.
func jsonFields(recordType reflect.Type, prefix string, path []int) []jsonField {
	var fields []jsonField
	for _, field := range reflect.VisibleFields(recordType) {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" || field.Type.Kind() == reflect.Slice {
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

// Cell is one human table cell: plain text plus an optional Paint applied
// after padding so alignment ignores escape codes.
type Cell struct {
	Paint func(string) string
	text  string
}

// Plain creates an unpainted table cell.
func Plain(text string) Cell { return Cell{text: text} }

// Column describes a human table column.
type Column struct {
	Name  string
	Right bool
}

// Table renders a header and rows as space-separated columns padded
// to the widest cell. The last column is never padded.
func Table(w io.Writer, columns []Column, rows [][]Cell) error {
	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = utf8.RuneCountInString(col.Name)
	}
	for _, row := range rows {
		for i, c := range row {
			widths[i] = max(widths[i], utf8.RuneCountInString(c.text))
		}
	}

	var out strings.Builder
	header := make([]Cell, len(columns))
	for i, col := range columns {
		header[i] = Plain(col.Name)
	}
	for _, row := range append([][]Cell{header}, rows...) {
		var line strings.Builder
		for i, c := range row {
			padding := strings.Repeat(" ", widths[i]-utf8.RuneCountInString(c.text))
			text := c.text
			if c.Paint != nil {
				text = c.Paint(text)
			}
			switch {
			case i == len(row)-1 && !columns[i].Right:
				line.WriteString(text)
			case columns[i].Right:
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

// Fields renders one record as aligned "name  value" lines. A value
// spanning several lines, such as a note, continues under the value
// column.
func Fields(w io.Writer, fields [][2]string) error {
	width := 0
	for _, field := range fields {
		width = max(width, utf8.RuneCountInString(field[0]))
	}
	var out strings.Builder
	for _, field := range fields {
		for i, line := range strings.Split(field[1], "\n") {
			name := field[0]
			if i > 0 {
				name = ""
			}
			out.WriteString(strings.TrimRight(fmt.Sprintf("%-*s  %s", width, name, line), " "))
			out.WriteByte('\n')
		}
	}
	_, err := io.WriteString(w, out.String())
	return err
}

// OptionalInt renders a nullable count for human output, blank when nil.
func OptionalInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

// CountNoun pluralizes a summary count.
func CountNoun(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}
