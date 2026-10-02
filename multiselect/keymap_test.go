package multiselect

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func runeKey(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func TestRebindDown(t *testing.T) {
	m := NewStrings("a", "b", "c")
	m.KeyMap.Down.Keys = []string{"j"}
	m, _ = m.Update(key(tui.KeyDown))
	if m.Cursor() != 0 {
		t.Fatalf("arrow moved after rebind: %d", m.Cursor())
	}
	m, _ = m.Update(runeKey('j'))
	if m.Cursor() != 1 {
		t.Fatalf("j did not move: %d", m.Cursor())
	}
}

func TestZeroKeyMapFallsBack(t *testing.T) {
	m := Model{Items: []Item{{Label: "a"}, {Label: "b"}}}
	m, _ = m.Update(key(tui.KeyDown))
	if m.Cursor() != 1 {
		t.Fatalf("literal Model: cursor %d, want 1", m.Cursor())
	}
}

func TestBindings(t *testing.T) {
	bs := NewStrings("a").Bindings()
	if len(bs) == 0 {
		t.Fatal("no bindings")
	}
	for _, b := range bs {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("incomplete binding %+v", b)
		}
	}
}

func click(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
}

func TestMouseClickSelectsRow(t *testing.T) {
	m := NewStrings("a", "b", "c", "d", "e")
	m.Bounds = hittest.Rect{X: 5, Y: 2, W: 10, H: 5}
	m, _ = m.Update(click(6, 2+3))
	if m.Cursor() != 0 {
		t.Fatalf("Mouse off: cursor %d, want 0", m.Cursor())
	}
	m.Mouse = true
	m, _ = m.Update(click(6, 2+3))
	if m.Cursor() != 3 {
		t.Fatalf("cursor %d, want 3", m.Cursor())
	}
	m, _ = m.Update(click(0, 0)) // outside Bounds
	if m.Cursor() != 3 {
		t.Fatalf("outside click moved cursor to %d", m.Cursor())
	}
}

func TestMouseWheelStep(t *testing.T) {
	m := NewStrings("a", "b", "c", "d", "e", "f", "g", "h")
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 10, H: 8}
	wheel := func(b tui.MouseButton) tui.MouseEvent {
		return tui.MouseEvent{X: 1, Y: 1, Button: b, Action: tui.MouseActionPress}
	}
	m, _ = m.Update(wheel(tui.MouseButtonWheelDown))
	if m.Cursor() != 3 {
		t.Fatalf("default step: cursor %d, want 3", m.Cursor())
	}
	m.WheelStep = 2
	m, _ = m.Update(wheel(tui.MouseButtonWheelDown))
	if m.Cursor() != 5 {
		t.Fatalf("step 2: cursor %d, want 5", m.Cursor())
	}
	m, _ = m.Update(wheel(tui.MouseButtonWheelUp))
	m, _ = m.Update(wheel(tui.MouseButtonWheelUp))
	m, _ = m.Update(wheel(tui.MouseButtonWheelUp))
	if m.Cursor() != 0 {
		t.Fatalf("clamp: cursor %d, want 0", m.Cursor())
	}
}

func TestRebindToggle(t *testing.T) {
	m := NewStrings("a", "b")
	m.KeyMap.Toggle.Keys = []string{"x"}
	m, _ = m.Update(key(tui.KeySpace))
	if m.IsSelected(0) {
		t.Fatal("space toggled after rebind")
	}
	m, _ = m.Update(runeKey(0x78))
	if !m.IsSelected(0) {
		t.Fatal("x did not toggle")
	}
}
