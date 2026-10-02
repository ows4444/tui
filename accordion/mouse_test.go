package accordion

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func press(x, y int, b tui.MouseButton) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: tui.MouseActionPress}
}

func mouseModel() Model {
	m := New(Section{"A", "a1\na2"}, Section{"B", "b1"}, Section{"C", ""})
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 5, Y: 2, W: 20, H: 10}
	return m
}

func TestClickHeaderMovesCursorAndToggles(t *testing.T) {
	m := mouseModel()
	m, _ = m.Update(press(6, 2, tui.MouseButtonLeft)) // header A
	if !m.IsExpanded(0) || m.Cursor() != 0 {
		t.Fatalf("A: expanded=%v cursor=%d", m.IsExpanded(0), m.Cursor())
	}
	// A is expanded: rows 2 A, 3-4 content, 5 B, 6 B content, 7 C.
	m, _ = m.Update(press(6, 5, tui.MouseButtonLeft))
	if !m.IsExpanded(1) || m.Cursor() != 1 {
		t.Fatalf("B: expanded=%v cursor=%d", m.IsExpanded(1), m.Cursor())
	}
	m, _ = m.Update(press(6, 7, tui.MouseButtonLeft))
	if m.Cursor() != 2 || !m.IsExpanded(2) {
		t.Fatalf("C after expansion shifted rows: cursor=%d", m.Cursor())
	}
	m, _ = m.Update(press(6, 2, tui.MouseButtonLeft))
	if m.IsExpanded(0) {
		t.Error("second click did not collapse A")
	}
}

func TestClickOnContentOrBelowOrOutsideDoesNothing(t *testing.T) {
	m := mouseModel()
	m.Toggle(0)
	for _, ev := range []tui.MouseEvent{
		press(6, 3, tui.MouseButtonLeft),  // content line
		press(6, 11, tui.MouseButtonLeft), // below last section
		press(0, 2, tui.MouseButtonLeft),  // left of Bounds
		press(6, 2, tui.MouseButtonRight), // wrong button
		{X: 6, Y: 2, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease},
	} {
		got, _ := m.Update(ev)
		if got.Cursor() != 0 || !got.IsExpanded(0) || got.IsExpanded(1) || got.IsExpanded(2) {
			t.Errorf("%+v changed state", ev)
		}
	}
}

func TestWheelMovesCursor(t *testing.T) {
	m := mouseModel()
	m, _ = m.Update(press(6, 3, tui.MouseButtonWheelDown))
	m, _ = m.Update(press(6, 3, tui.MouseButtonWheelDown))
	m, _ = m.Update(press(6, 3, tui.MouseButtonWheelDown))
	if m.Cursor() != 2 {
		t.Errorf("cursor = %d, want 2 (clamped)", m.Cursor())
	}
	m, _ = m.Update(press(6, 3, tui.MouseButtonWheelUp))
	if m.Cursor() != 1 {
		t.Errorf("cursor = %d, want 1", m.Cursor())
	}
}

func TestMouseOffIgnoresEvents(t *testing.T) {
	m := mouseModel()
	m.Mouse = false
	got, _ := m.Update(press(6, 2, tui.MouseButtonLeft))
	if got.IsExpanded(0) {
		t.Error("mouse handled while Mouse is off")
	}
}
