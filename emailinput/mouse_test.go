package emailinput

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func ev(x, y int, b tui.MouseButton, a tui.MouseAction) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: a}
}

func mouseModel() Model {
	m := New()
	m.SetValue("abcdef")
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 4, Y: 2, W: 20, H: 1}
	return m
}

func TestClickMovesCursor(t *testing.T) {
	m, _ := mouseModel().Update(ev(4+2, 2, tui.MouseButtonLeft, tui.MouseActionPress))
	if m.Cursor() != 2 {
		t.Fatalf("cursor = %d, want 2", m.Cursor())
	}
}

func TestClickIgnored(t *testing.T) {
	base := mouseModel()
	base.SetCursor(6)
	for name, e := range map[string]tui.MouseEvent{
		"miss":    ev(0, 2, tui.MouseButtonLeft, tui.MouseActionPress),
		"button":  ev(6, 2, tui.MouseButtonRight, tui.MouseActionPress),
		"release": ev(6, 2, tui.MouseButtonLeft, tui.MouseActionRelease),
	} {
		got, _ := base.Update(e)
		if got.Cursor() != 6 {
			t.Errorf("%s: cursor = %d, want 6", name, got.Cursor())
		}
	}
}

func TestMouseOff(t *testing.T) {
	m := mouseModel()
	m.Mouse = false
	m.SetCursor(6)
	got, _ := m.Update(ev(6, 2, tui.MouseButtonLeft, tui.MouseActionPress))
	if got.Cursor() != 6 {
		t.Fatalf("cursor = %d, want 6", got.Cursor())
	}
}
