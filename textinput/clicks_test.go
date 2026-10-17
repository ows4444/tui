package textinput

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func clickAt(m Model, x, clicks int) Model {
	m, _ = m.Update(tui.MouseEvent{X: x, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: clicks})
	m, _ = m.Update(tui.MouseEvent{X: x, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease, Clicks: clicks})
	return m
}

func clicky() Model {
	m := New()
	m.Prompt = ""
	m.SetValue("alpha beta  gamma")
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 30, H: 1}
	m.Focus()
	return m
}

func TestDoubleClickSelectsTheWord(t *testing.T) {
	m := clickAt(clicky(), 8, 2) // inside "beta"
	if got := m.ed.SelectedText(); got != "beta" {
		t.Fatalf("double click selected %q; want beta", got)
	}
	m = clickAt(clicky(), 10, 2) // in the gap
	if got := m.ed.SelectedText(); got != "  " {
		t.Fatalf("double click on the gap selected %q", got)
	}
}

func TestTripleClickSelectsEverything(t *testing.T) {
	m := clickAt(clicky(), 2, 3)
	if got := m.ed.SelectedText(); got != "alpha beta  gamma" {
		t.Fatalf("triple click selected %q", got)
	}
}

func TestSingleClickStillJustPlacesTheCursor(t *testing.T) {
	for _, c := range []int{0, 1} {
		m := clickAt(clicky(), 8, c)
		if m.ed.SelectedText() != "" || m.ed.Cursor() != 8 {
			t.Fatalf("Clicks=%d: selection %q cursor %d", c, m.ed.SelectedText(), m.ed.Cursor())
		}
	}
}

func TestDoubleClickThenDragDoesNotShrinkTheWord(t *testing.T) {
	m := clicky()
	m, _ = m.Update(tui.MouseEvent{X: 8, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: 2})
	m, _ = m.Update(tui.MouseEvent{X: 14, Button: tui.MouseButtonLeft, Action: tui.MouseActionMotion, Clicks: 2})
	if got := m.ed.SelectedText(); got != "beta" {
		t.Fatalf("a drag after a double click changed the selection to %q", got)
	}
}
