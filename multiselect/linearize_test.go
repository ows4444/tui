package multiselect

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := NewStrings("Milk", "Eggs", "Bread")
	m.Toggle(0)
	m.Toggle(2)
	m.SetCursor(1)
	want := "Milk, checked, item 1 of 3\nEggs, not checked, item 2 of 3, current\nBread, checked, item 3 of 3"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	if strings.ContainsAny(m.Linearize(), "\x1b[]>") {
		t.Errorf("Linearize has escapes or glyphs: %q", m.Linearize())
	}
	if got := NewStrings().Linearize(); got != "No items" {
		t.Errorf("empty = %q", got)
	}
}
