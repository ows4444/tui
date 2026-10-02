package virtuallist

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearizeRendersOnlyTheVisibleWindow(t *testing.T) {
	calls := 0
	m := New(10000, 3, func(i int) string { calls++; return fmt.Sprintf("row %d\nmore", i) })
	m.LineDown(100)
	calls = 0
	got := m.Linearize()
	want := "List, 10000 items, showing 101 to 103\nItem 101 of 10000: row 100 more\nItem 102 of 10000: row 101 more\nItem 103 of 10000: row 102 more"
	if got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	if calls > 3 {
		t.Errorf("rendered %d items for a 3-row window", calls)
	}
	if got := New(0, 3, func(int) string { return "" }).Linearize(); got != "List, empty" {
		t.Errorf("empty = %q", got)
	}
	if strings.Contains(New(5, 2, func(i int) string { return "\x1b[1mx\x1b[0m" }).Linearize(), "\x1b") {
		t.Error("escapes must be stripped")
	}
}
