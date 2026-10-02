package tabs

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func runeKey(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func TestRebindNext(t *testing.T) {
	m := New("a", "b", "c")
	m.KeyMap.Next.Keys = []string{"l"}
	m, _ = m.Update(key(tui.KeyRight))
	if m.Active() != 0 {
		t.Fatalf("arrow moved after rebind: %d", m.Active())
	}
	m, _ = m.Update(runeKey('l'))
	if m.Active() != 1 {
		t.Fatalf("l did not move: %d", m.Active())
	}
}

func TestZeroKeyMapFallsBack(t *testing.T) {
	m := Model{Labels: []string{"a", "b"}}
	m, _ = m.Update(key(tui.KeyRight))
	if m.Active() != 1 {
		t.Fatalf("literal Model: active %d, want 1", m.Active())
	}
}

func TestBindings(t *testing.T) {
	bs := New("a").Bindings()
	if len(bs) == 0 {
		t.Fatal("no bindings")
	}
	for _, b := range bs {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("incomplete binding %+v", b)
		}
	}
}

func TestMouseClickSelectsTab(t *testing.T) {
	m := New("one", "two", "three")
	m.Bounds = hittest.Rect{X: 4, Y: 1, W: 30, H: 1}
	click := func(x, y int) tui.MouseEvent {
		return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
	}
	// " one " is cols 0-4, gap 5, " two " 6-10, gap 11, " three " 12-18.
	m, _ = m.Update(click(4+13, 1))
	if m.Active() != 0 {
		t.Fatalf("Mouse off: active %d, want 0", m.Active())
	}
	m.Mouse = true
	m, cmd := m.Update(click(4+13, 1))
	if m.Active() != 2 {
		t.Fatalf("active %d, want 2", m.Active())
	}
	if cmd == nil {
		t.Fatal("no ChangedMsg cmd")
	}
	if msg, _ := cmd().(ChangedMsg); msg.Index != 2 {
		t.Fatalf("ChangedMsg %+v", msg)
	}
	m, _ = m.Update(click(4+7, 1))
	if m.Active() != 1 {
		t.Fatalf("active %d, want 1", m.Active())
	}
	m, _ = m.Update(click(4+11, 1)) // the gap
	m, _ = m.Update(click(4+1, 2))  // below the bar
	if m.Active() != 1 {
		t.Fatalf("gap/outside click changed active to %d", m.Active())
	}
}
