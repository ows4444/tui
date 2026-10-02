package spinner

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New()
	m.Label = "Loading data"
	if got := m.Linearize(); got != "Loading data, stopped" {
		t.Errorf("stopped = %q", got)
	}
	m.running = true
	if got := m.Linearize(); got != "Loading data, in progress" {
		t.Errorf("running = %q", got)
	}
	m.frame = 3
	if got := m.Linearize(); got != "Loading data, in progress" {
		t.Errorf("must be stable while it spins: %q", got)
	}
	bare := New()
	bare.running = true
	if got := bare.Linearize(); got != "Working, in progress" {
		t.Errorf("no label = %q", got)
	}
}
