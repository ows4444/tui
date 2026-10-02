package popover

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New("Tip: press ? for help", 2, 3)
	m.Hide()
	if got := m.Linearize(); got != "" {
		t.Errorf("closed = %q", got)
	}
	m.Show()
	if got := m.Linearize(); got != "Popover\nTip: press ? for help" {
		t.Errorf("open = %q", got)
	}
	empty := New("", 0, 0)
	empty.Show()
	if got := empty.Linearize(); got != "Popover" {
		t.Errorf("empty content = %q", got)
	}
}
