package helpscreen

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/widgets"
)

func ev(x, y int, b tui.MouseButton, a tui.MouseAction) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: a}
}

func mouseModel() Model {
	m := New(widgets.Hint{Key: "q", Action: "quit"})
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 40, H: 5}
	return m
}

func TestClickOutsideDismisses(t *testing.T) {
	m, cmd := mouseModel().Update(ev(10, 8, tui.MouseButtonLeft, tui.MouseActionPress))
	if m.Open() || cmd == nil {
		t.Fatal("click outside did not dismiss")
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Fatal("not DismissedMsg")
	}
}

func TestZeroBoundsAnyClickDismisses(t *testing.T) {
	m := mouseModel()
	m.Bounds = hittest.Rect{}
	if m, _ = m.Update(ev(1, 1, tui.MouseButtonLeft, tui.MouseActionPress)); m.Open() {
		t.Fatal("still open")
	}
}

func TestClickIgnored(t *testing.T) {
	for name, e := range map[string]tui.MouseEvent{
		"inside":  ev(5, 2, tui.MouseButtonLeft, tui.MouseActionPress),
		"button":  ev(10, 8, tui.MouseButtonRight, tui.MouseActionPress),
		"wheel":   ev(10, 8, tui.MouseButtonWheelDown, tui.MouseActionPress),
		"release": ev(10, 8, tui.MouseButtonLeft, tui.MouseActionRelease),
	} {
		m, cmd := mouseModel().Update(e)
		if !m.Open() || cmd != nil {
			t.Errorf("%s: changed state", name)
		}
	}
}

func TestMouseOff(t *testing.T) {
	m := mouseModel()
	m.Mouse = false
	if m, _ = m.Update(ev(10, 8, tui.MouseButtonLeft, tui.MouseActionPress)); !m.Open() {
		t.Fatal("Mouse off still dismissed")
	}
}
