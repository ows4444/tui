package taginput

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/widgets"
)

func ev(x, y int, b tui.MouseButton, a tui.MouseAction) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: a}
}

func mouseModel() (Model, int) {
	m := New()
	m.Tags = []string{"go", "tui"}
	m.Input.SetValue("abcdef")
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 3, Y: 1, W: 60, H: 1}
	used := 0
	for _, tag := range m.Tags {
		used += ansi.Width(widgets.Tag(tag, widgets.TagSolid, m.Variant, m.Theme)) + 1
	}
	return m, used
}

func TestClickInInputMovesCursor(t *testing.T) {
	m, used := mouseModel()
	m, _ = m.Update(ev(3+used+3, 1, tui.MouseButtonLeft, tui.MouseActionPress))
	if m.Input.Cursor() != 3 {
		t.Fatalf("cursor = %d, want 3", m.Input.Cursor())
	}
	if m.Input.Mouse || m.Input.Bounds != (hittest.Rect{}) {
		t.Error("Input's own Mouse/Bounds were changed")
	}
}

func TestClickIgnored(t *testing.T) {
	base, used := mouseModel()
	base.Input.SetCursor(6)
	for name, e := range map[string]tui.MouseEvent{
		"on chips": ev(3+1, 1, tui.MouseButtonLeft, tui.MouseActionPress),
		"miss":     ev(3+used+1, 5, tui.MouseButtonLeft, tui.MouseActionPress),
		"button":   ev(3+used+1, 1, tui.MouseButtonRight, tui.MouseActionPress),
		"release":  ev(3+used+1, 1, tui.MouseButtonLeft, tui.MouseActionRelease),
	} {
		got, _ := base.Update(e)
		if got.Input.Cursor() != 6 {
			t.Errorf("%s: cursor = %d, want 6", name, got.Input.Cursor())
		}
	}
}

func TestMouseOff(t *testing.T) {
	m, used := mouseModel()
	m.Mouse = false
	m.Input.SetCursor(6)
	got, _ := m.Update(ev(3+used+1, 1, tui.MouseButtonLeft, tui.MouseActionPress))
	if got.Input.Cursor() != 6 {
		t.Fatalf("cursor = %d, want 6", got.Input.Cursor())
	}
}
