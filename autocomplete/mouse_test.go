package autocomplete

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func ev(x, y int, b tui.MouseButton, a tui.MouseAction) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: a}
}

// Input "a" matches apple and avocado: rows 3 (input), 4 apple, 5 avocado.
func mouseModel() Model {
	m := New("apple", "avocado", "banana")
	m.Input.SetValue("a")
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 2, Y: 3, W: 20, H: 4}
	return m
}

func TestClickSuggestionAccepts(t *testing.T) {
	m, cmd := mouseModel().Update(ev(4, 5, tui.MouseButtonLeft, tui.MouseActionPress))
	if m.Input.Value() != "avocado" || m.IsOpen() || cmd == nil {
		t.Fatalf("value=%q open=%v", m.Input.Value(), m.IsOpen())
	}
	if a, ok := cmd().(AcceptedMsg); !ok || a.Value != "avocado" {
		t.Fatalf("msg %+v", a)
	}
}

func TestWheelMovesHighlight(t *testing.T) {
	m := mouseModel()
	m, _ = m.Update(ev(4, 5, tui.MouseButtonWheelDown, tui.MouseActionPress))
	if m.highlight != 1 {
		t.Fatalf("highlight = %d, want 1", m.highlight)
	}
	m, _ = m.Update(ev(4, 5, tui.MouseButtonWheelDown, tui.MouseActionPress)) // clamped
	if m.highlight != 1 {
		t.Fatalf("highlight = %d after clamp", m.highlight)
	}
	m, _ = m.Update(ev(4, 5, tui.MouseButtonWheelUp, tui.MouseActionPress))
	if m.highlight != 0 {
		t.Fatalf("highlight = %d, want 0", m.highlight)
	}
}

func TestClickInputRowPlacesCursor(t *testing.T) {
	m := mouseModel()
	m.Input.SetValue("ap")
	m, _ = m.Update(ev(2, 3, tui.MouseButtonLeft, tui.MouseActionPress))
	if m.Input.Cursor() != 0 || m.Input.Value() != "ap" || m.Input.Mouse {
		t.Fatalf("cursor=%d value=%q inputMouse=%v", m.Input.Cursor(), m.Input.Value(), m.Input.Mouse)
	}
}

func TestIgnored(t *testing.T) {
	for name, e := range map[string]tui.MouseEvent{
		"miss":       ev(40, 5, tui.MouseButtonLeft, tui.MouseActionPress),
		"below list": ev(4, 6, tui.MouseButtonLeft, tui.MouseActionPress),
		"button":     ev(4, 5, tui.MouseButtonRight, tui.MouseActionPress),
		"release":    ev(4, 5, tui.MouseButtonLeft, tui.MouseActionRelease),
		"wheel miss": ev(40, 5, tui.MouseButtonWheelDown, tui.MouseActionPress),
	} {
		m, cmd := mouseModel().Update(e)
		if m.Input.Value() != "a" || m.highlight != 0 || cmd != nil {
			t.Errorf("%s: changed state", name)
		}
	}
}

func TestMouseOff(t *testing.T) {
	m := mouseModel()
	m.Mouse = false
	m, cmd := m.Update(ev(4, 5, tui.MouseButtonLeft, tui.MouseActionPress))
	if m.Input.Value() != "a" || cmd != nil {
		t.Fatal("Mouse off still accepted")
	}
}
