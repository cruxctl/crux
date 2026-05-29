package output

import (
	"bytes"
	"testing"
)

type row struct {
	Name  string `json:"name"  yaml:"name"`
	Count int    `json:"count" yaml:"count"`
}

func TestRender_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, FormatJSON, []row{{Name: "a", Count: 1}})
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	want := "[{\"name\":\"a\",\"count\":1}]\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestRender_Table(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, FormatTable, []row{{Name: "a", Count: 1}, {Name: "bb", Count: 22}})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if len(out) == 0 {
		t.Errorf("empty table")
	}
	// Header + rows
	if !contains(out, "NAME") || !contains(out, "COUNT") || !contains(out, "bb") {
		t.Errorf("table missing expected fields: %q", out)
	}
}

func contains(haystack, needle string) bool {
	return bytes.Contains([]byte(haystack), []byte(needle))
}
