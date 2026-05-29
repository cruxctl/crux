// Package output renders Go structs as table, json, or yaml. All CLI command
// handlers MUST funnel terminal output through this package; no direct fmt
// printing of structured data.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

// Render writes v to w as the given format. v must be a slice or a single
// struct. For tables, struct fields become columns in declaration order.
func Render(w io.Writer, f Format, v any) error {
	switch f {
	case FormatJSON:
		return json.NewEncoder(w).Encode(v)
	case FormatYAML:
		return yaml.NewEncoder(w).Encode(v)
	case FormatTable, "":
		return renderTable(w, v)
	default:
		return fmt.Errorf("unknown output format: %q", f)
	}
}

func renderTable(w io.Writer, v any) error {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice {
		return renderRow(w, rv)
	}
	if rv.Len() == 0 {
		_, err := fmt.Fprintln(w, "(no results)")
		return err
	}
	first := rv.Index(0)
	for first.Kind() == reflect.Ptr {
		first = first.Elem()
	}
	if first.Kind() != reflect.Struct {
		// scalar slice — print one per line.
		for i := 0; i < rv.Len(); i++ {
			fmt.Fprintln(w, rv.Index(i).Interface())
		}
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	t := first.Type()
	headers := make([]string, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		headers[i] = strings.ToUpper(t.Field(i).Name)
	}
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	for i := 0; i < rv.Len(); i++ {
		row := rv.Index(i)
		for row.Kind() == reflect.Ptr {
			row = row.Elem()
		}
		cells := make([]string, row.NumField())
		for j := 0; j < row.NumField(); j++ {
			cells[j] = fmt.Sprint(row.Field(j).Interface())
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	return tw.Flush()
}

func renderRow(w io.Writer, rv reflect.Value) error {
	if rv.Kind() != reflect.Struct {
		_, err := fmt.Fprintln(w, rv.Interface())
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	t := rv.Type()
	for i := 0; i < t.NumField(); i++ {
		fmt.Fprintf(tw, "%s\t%v\n", strings.ToUpper(t.Field(i).Name), rv.Field(i).Interface())
	}
	return tw.Flush()
}
