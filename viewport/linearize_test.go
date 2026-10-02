package viewport

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearizeIsTheWholeContentWithoutEscapes(t *testing.T) {
	m := New(5, 2) // a tiny window: Linearize must not window
	m.SetContent("one\n\x1b[1mtwo\x1b[0m\nthree")
	m.LineDown(1)
	if got := m.Linearize(); got != "one\ntwo\nthree" {
		t.Errorf("Linearize = %q", got)
	}
	if got := New(5, 2).Linearize(); got != "" {
		t.Errorf("no content = %q", got)
	}
}
