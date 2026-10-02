package accordion

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New(
		Section{Title: "Display", Content: "Theme: Dark\nFont: 14"},
		Section{Title: "Sound", Content: "On"},
	)
	want := "Display, section 1 of 2, collapsed, selected\nSound, section 2 of 2, collapsed"
	if got := m.Linearize(); got != want {
		t.Errorf("collapsed =\n%s\nwant\n%s", got, want)
	}
	m.Toggle(0)
	want = "Display, section 1 of 2, expanded, selected\nDisplay content: Theme: Dark\nDisplay content: Font: 14\nSound, section 2 of 2, collapsed"
	if got := m.Linearize(); got != want {
		t.Errorf("expanded =\n%s\nwant\n%s", got, want)
	}
	if got := New().Linearize(); got != "No sections" {
		t.Errorf("empty = %q", got)
	}
}
