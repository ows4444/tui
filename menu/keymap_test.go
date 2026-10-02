package menu

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func kt(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func runeKey(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func sample() Model {
	return New([]Item{
		{Label: "a", Value: "a", Children: []Item{{Label: "a1", Value: "a1"}, {Label: "a2", Value: "a2"}}},
		{Label: "b", Value: "b"},
		{Label: "c", Value: "c"},
		{Label: "d", Value: "d"},
		{Label: "e", Value: "e"},
	})
}

func cursor(m Model) int { return m.stack[m.current()].Cursor() }

func TestRebindDown(t *testing.T) {
	m := sample()
	m.KeyMap.Down.Keys = []string{"j"}
	m, _ = m.Update(kt(tui.KeyDown))
	if cursor(m) != 0 {
		t.Fatalf("arrow moved after rebind: %d", cursor(m))
	}
	m, _ = m.Update(runeKey('j'))
	if cursor(m) != 1 {
		t.Fatalf("j did not move: %d", cursor(m))
	}
}

func TestRebindBack(t *testing.T) {
	m := sample()
	m.KeyMap.Back.Keys = []string{"h"}
	m, _ = m.Update(kt(tui.KeyEnter)) // drill into a
	m, _ = m.Update(kt(tui.KeyEsc))
	if m.current() != 1 {
		t.Fatal("esc popped after rebind")
	}
	m, _ = m.Update(runeKey('h'))
	if m.current() != 0 {
		t.Fatal("h did not pop")
	}
}

func TestZeroKeyMapFallsBack(t *testing.T) {
	m := sample()
	m.KeyMap = KeyMap{}
	m, _ = m.Update(kt(tui.KeyDown))
	if cursor(m) != 1 {
		t.Fatalf("cursor %d, want 1", cursor(m))
	}
}

func TestBindings(t *testing.T) {
	bs := sample().Bindings()
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
	m := sample()
	m.Bounds = hittest.Rect{X: 2, Y: 4, W: 10, H: 5}
	click := tui.MouseEvent{X: 3, Y: 4 + 3, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
	m, _ = m.Update(click)
	if cursor(m) != 0 {
		t.Fatalf("Mouse off: cursor %d, want 0", cursor(m))
	}
	m.Mouse = true
	m, _ = m.Update(click)
	if cursor(m) != 3 {
		t.Fatalf("cursor %d, want 3", cursor(m))
	}
}

func TestMouseWheelStep(t *testing.T) {
	m := sample()
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 10, H: 5}
	m.WheelStep = 2
	down := tui.MouseEvent{X: 1, Y: 1, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress}
	m, _ = m.Update(down)
	if cursor(m) != 2 {
		t.Fatalf("cursor %d, want 2", cursor(m))
	}
	m.WheelStep = 0 // default 3, clamped at the last row
	m, _ = m.Update(down)
	if cursor(m) != 4 {
		t.Fatalf("cursor %d, want 4", cursor(m))
	}
}
