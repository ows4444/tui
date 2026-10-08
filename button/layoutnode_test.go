package button

import (
	"testing"

	"github.com/ows4444/tui/layout"
)

func TestLayoutNodeIsOneRowOfTheButtonsWidth(t *testing.T) {
	m := New("Save")
	n := m.LayoutNode()
	s := n.Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded})
	if s != (layout.Size{W: m.Width(), H: 1}) {
		t.Fatalf("Measure = %+v, want %dx1", s, m.Width())
	}
	if got := n.Render(s); got != m.View() {
		t.Fatalf("Render = %q, want View %q", got, m.View())
	}
}
