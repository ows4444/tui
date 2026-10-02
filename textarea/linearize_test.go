package textarea

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New()
	m.Placeholder = "notes"
	if got := m.Linearize(); got != "Text area, empty, placeholder: notes" {
		t.Errorf("empty = %q", got)
	}
	m.SetValue("first\n\nthird")
	want := "Text area, 3 lines\nLine 1 of 3: first\nLine 2 of 3: blank\nLine 3 of 3: third"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	m.Focus()
	m.SetCursor(2) // line 1, column 3
	if got := strings.Split(m.Linearize(), "\n")[0]; got != "Text area, 3 lines, focused, cursor on line 1, column 3" {
		t.Errorf("focused header = %q", got)
	}
	one := New()
	one.SetValue("only")
	if got := strings.Split(one.Linearize(), "\n")[0]; got != "Text area, 1 line" {
		t.Errorf("singular = %q", got)
	}
}
