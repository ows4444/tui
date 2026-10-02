package helpscreen

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/widgets"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New(widgets.Hint{Key: "q", Action: "quit"}, widgets.Hint{Key: "?", Action: "toggle help"})
	m.Hide()
	if got := m.Linearize(); got != "" {
		t.Errorf("closed = %q", got)
	}
	m.open = true
	if got := m.Linearize(); got != "Help, 2 key bindings\nKey: q, quit\nKey: ?, toggle help" {
		t.Errorf("open = %q", got)
	}
	one := New(widgets.Hint{Key: "q", Action: "quit"})
	one.open = true
	if got := one.Linearize(); got != "Help, 1 key binding\nKey: q, quit" {
		t.Errorf("singular = %q", got)
	}
}
