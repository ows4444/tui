package loadingbar

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New(10)
	if got := m.Linearize(); got != "Loading, stopped" {
		t.Errorf("idle = %q", got)
	}
	m.Start()
	if got := m.Linearize(); got != "Loading, in progress" {
		t.Errorf("running = %q", got)
	}
	m.Stop()
	if got := m.Linearize(); got != "Loading, stopped" {
		t.Errorf("stopped = %q", got)
	}
	if got := New(0).Linearize(); got != "" {
		t.Errorf("zero width = %q", got)
	}
}
