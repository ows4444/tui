package tabs

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New("Home", "Logs", "Settings")
	m.SetActive(1)
	want := "Home, tab 1 of 3\nLogs, tab 2 of 3, selected\nSettings, tab 3 of 3"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
	if got := New().Linearize(); got != "No tabs" {
		t.Errorf("empty = %q", got)
	}
}
