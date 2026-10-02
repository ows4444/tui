package drawer

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New("Settings\nTheme: dark")
	m.Hide()
	if got := m.Linearize(); got != "" {
		t.Errorf("closed = %q", got)
	}
	m.Show()
	if got := m.Linearize(); got != "Drawer, right edge\nSettings\nTheme: dark" {
		t.Errorf("open = %q", got)
	}
	m.Edge = EdgeBottom
	if got := m.Linearize(); got != "Drawer, bottom edge\nSettings\nTheme: dark" {
		t.Errorf("bottom = %q", got)
	}
}
