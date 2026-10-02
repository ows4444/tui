package skeleton

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New()
	m.Width, m.Lines = 20, 3
	if got := m.Linearize(); got != "Loading placeholder" {
		t.Errorf("Linearize = %q", got)
	}
	m.frame = 7
	if got := m.Linearize(); got != "Loading placeholder" {
		t.Errorf("must not depend on the animation frame: %q", got)
	}
	if got := New().Linearize(); got != "" {
		t.Errorf("no size = %q", got)
	}
}
