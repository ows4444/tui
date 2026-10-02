package logview

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearizeGrowsOnlyAtTheEnd(t *testing.T) {
	m := New(20, 2)
	m.Append("first")
	m.Append("\x1b[31msecond\x1b[0m")
	a := m.Linearize()
	if a != "first\nsecond" {
		t.Errorf("Linearize = %q", a)
	}
	m.Append("third")
	b := m.Linearize()
	if b != a+"\nthird" {
		t.Errorf("an append must only add a line at the end (append-only transcript): %q -> %q", a, b)
	}
}
