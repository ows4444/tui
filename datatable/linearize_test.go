package datatable

import (
	"strings"
	"testing"
)

func TestLinearize(t *testing.T) {
	m := New([]string{"Name", "Status"}, [][]string{{"a", "ok"}, {"b", "down"}, {"c", "ok", "extra"}})
	m.SetCursor(1)
	want := "Row 1 of 3: Name: a, Status: ok\n" +
		"Row 2 of 3: Name: b, Status: down, selected\n" +
		"Row 3 of 3: Name: c, Status: ok, Column 3: extra"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	if strings.Contains(m.Linearize(), "\x1b") {
		t.Error("Linearize must not contain escape sequences")
	}
	if got := New([]string{"A"}, nil).Linearize(); got != "No rows" {
		t.Errorf("empty table = %q", got)
	}
}
