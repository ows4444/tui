package picker

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := NewStrings("Red", "Green", "Blue")
	m.SetCursor(1)
	want := "Red, item 1 of 3\nGreen, item 2 of 3, selected\nBlue, item 3 of 3"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	if strings.Contains(m.Linearize(), "\x1b") || strings.Contains(m.Linearize(), ">") {
		t.Error("Linearize must have no escapes or cursor glyph")
	}
	if got := NewStrings().Linearize(); got != "No items" {
		t.Errorf("empty = %q", got)
	}
}
