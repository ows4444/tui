package logview

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

// A log is selectable with the mouse through its viewport: set Viewport.Mouse
// and Viewport.Bounds, and a drag over the rows selects their text.
func TestLogSelectionThroughTheViewport(t *testing.T) {
	m := New(20, 3)
	for _, l := range []string{"first line", "second line", "third line"} {
		m.Append(l)
	}
	m.Viewport.Mouse = true
	m.Viewport.Bounds = hittest.Rect{X: 2, Y: 4, W: 20, H: 3}
	ev := func(a tui.MouseAction, x, y int) tui.MouseEvent {
		return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: a}
	}
	m, _ = m.Update(ev(tui.MouseActionPress, 2, 4))
	m, _ = m.Update(ev(tui.MouseActionMotion, 8, 5))
	m, _ = m.Update(ev(tui.MouseActionRelease, 8, 5))
	if got := m.Viewport.SelectedText(); got != "first line\nsecond " {
		t.Fatalf("selected %q", got)
	}
}
