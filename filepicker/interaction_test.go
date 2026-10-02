package filepicker

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func TestKeyMap_RebindDownToJ(t *testing.T) {
	m := manyEntries(10)
	m.KeyMap.Down.Keys = []string{"j"}
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	if m.Cursor() != 0 {
		t.Fatalf("arrow moved after rebind: cursor=%d", m.Cursor())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "j"})
	if m.Cursor() != 1 {
		t.Fatalf("cursor=%d, want 1", m.Cursor())
	}
}

func TestKeyMap_ZeroLiteralAndBindings(t *testing.T) {
	m := manyEntries(10)
	m.KeyMap = KeyMap{}
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	if m.Cursor() != 1 {
		t.Fatalf("cursor=%d, want 1", m.Cursor())
	}
	for _, b := range m.Bindings() {
		if len(b.Keys) == 0 || b.Desc == "" {
			t.Fatalf("incomplete binding %+v", b)
		}
	}
}

func TestMouse_ClickSelectsRowAndWheelScrolls(t *testing.T) {
	m := manyEntries(100)
	m.Height = 10
	m.Bounds = hittest.Rect{X: 0, Y: 1, W: 30, H: 10}
	click := tui.MouseEvent{X: 2, Y: 4, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
	off, _ := m.Update(click)
	if off.Cursor() != 0 {
		t.Fatal("mouse off must ignore events")
	}
	m.Mouse = true
	m, _ = m.Update(click)
	if m.Cursor() != 3 {
		t.Fatalf("cursor=%d, want 3", m.Cursor())
	}
	wheel := tui.MouseEvent{X: 2, Y: 4, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress}
	m, _ = m.Update(wheel)
	if s, _ := m.window(); s != 3 {
		t.Fatalf("window start=%d, want 3", s)
	}
	m.WheelStep = 5
	m, _ = m.Update(wheel)
	if s, _ := m.window(); s != 8 {
		t.Fatalf("window start=%d, want 8", s)
	}
	if m.Cursor() < 8 || m.Cursor() > 17 {
		t.Fatalf("cursor %d left the window", m.Cursor())
	}
	m, _ = m.Update(tui.MouseEvent{X: 2, Y: 4, Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if s, _ := m.window(); s != 3 {
		t.Fatalf("window start=%d, want 3", s)
	}
}
