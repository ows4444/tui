package streamtext

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearizeGivesAllTheTextAtOnce(t *testing.T) {
	m := NewTypewriter()
	m.SetText("hello world")
	if got := m.View(); got == "hello world" {
		t.Skip("reveal already complete; premise of the test does not hold")
	}
	if got := m.Linearize(); got != "hello world" {
		t.Errorf("Linearize = %q, want the full text without waiting for the reveal", got)
	}
	if got := New().Linearize(); got != "" {
		t.Errorf("empty = %q", got)
	}
}
