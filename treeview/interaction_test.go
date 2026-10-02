package treeview

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
)

func runeKey(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func flat(n int) []Node {
	nodes := make([]Node, n)
	for i := range nodes {
		nodes[i] = Node{Label: "n" + string(rune('a'+i))}
	}
	return nodes
}

func TestRebindDownToJ(t *testing.T) {
	m := New(flat(3)...)
	m.KeyMap.Down = keymap.NewBinding("down", "j")
	m, _ = m.Update(key(tui.KeyDown))
	if m.Cursor() != 0 {
		t.Fatalf("arrow moved after rebind: %d", m.Cursor())
	}
	m, _ = m.Update(runeKey('j'))
	if m.Cursor() != 1 {
		t.Fatalf("j did not move: %d", m.Cursor())
	}
}

func TestLiteralModelUsesDefaultKeyMap(t *testing.T) {
	m := Model{Roots: flat(3)}
	m, _ = m.Update(key(tui.KeyDown))
	if m.Cursor() != 1 {
		t.Fatalf("Cursor = %d, want 1", m.Cursor())
	}
}

func TestBindingsHaveDescriptions(t *testing.T) {
	bs := Model{}.Bindings()
	if len(bs) == 0 {
		t.Fatal("no bindings")
	}
	for _, b := range bs {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("incomplete binding %+v", b)
		}
	}
}

func TestMouseClickSelectsRow(t *testing.T) {
	m := New(flat(6)...)
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 2, Y: 3, W: 20, H: 6}
	m, _ = m.Update(tui.MouseEvent{X: 4, Y: 6, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Cursor() != 3 {
		t.Fatalf("Cursor = %d, want 3", m.Cursor())
	}
	m, _ = m.Update(tui.MouseEvent{X: 40, Y: 4, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Cursor() != 3 {
		t.Fatalf("click outside Bounds moved the cursor: %d", m.Cursor())
	}
	m.Mouse = false
	m, _ = m.Update(tui.MouseEvent{X: 4, Y: 3, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Cursor() != 3 {
		t.Fatalf("mouse off still selected: %d", m.Cursor())
	}
}

func TestMouseWheelStep(t *testing.T) {
	m := New(flat(20)...)
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 20, H: 10}
	wheel := func(b tui.MouseButton) tui.MouseEvent {
		return tui.MouseEvent{X: 1, Y: 1, Button: b, Action: tui.MouseActionPress}
	}
	m, _ = m.Update(wheel(tui.MouseButtonWheelDown))
	if m.Cursor() != 3 {
		t.Fatalf("default step: Cursor = %d, want 3", m.Cursor())
	}
	m.WheelStep = 2
	m, _ = m.Update(wheel(tui.MouseButtonWheelDown))
	if m.Cursor() != 5 {
		t.Fatalf("step 2: Cursor = %d, want 5", m.Cursor())
	}
	m, _ = m.Update(wheel(tui.MouseButtonWheelUp))
	if m.Cursor() != 3 {
		t.Fatalf("wheel up: Cursor = %d, want 3", m.Cursor())
	}
	m.WheelStep = 100
	m, _ = m.Update(wheel(tui.MouseButtonWheelDown))
	if m.Cursor() != 19 {
		t.Fatalf("clamp: Cursor = %d, want 19", m.Cursor())
	}
}
