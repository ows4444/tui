package dialog

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearizeIsEmptyWhenClosed(t *testing.T) {
	m := New("Delete file", "This cannot be undone.")
	m.Hide()
	if got := m.Linearize(); got != "" {
		t.Errorf("closed = %q", got)
	}
	m.Show()
	if got := m.Linearize(); got != "Dialog: Delete file\nThis cannot be undone." {
		t.Errorf("open = %q", got)
	}
	m2 := New("Just a title", "")
	m2.Show()
	if got := m2.Linearize(); got != "Dialog: Just a title" {
		t.Errorf("no message = %q", got)
	}
}
