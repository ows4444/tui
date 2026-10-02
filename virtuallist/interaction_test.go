package virtuallist

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func rk(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func TestKeyMap_RebindDownToJ(t *testing.T) {
	m := New(100, 10, func(i int) string { return "" })
	m.KeyMap.Down.Keys = []string{"j"}
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	if m.Offset() != 0 || m.Cursor() != 0 {
		t.Fatalf("arrow moved after rebind: offset=%d cursor=%d", m.Offset(), m.Cursor())
	}
	m, _ = m.Update(rk('j'))
	if m.Offset() != 1 || m.Cursor() != 1 {
		t.Fatalf("j: offset=%d cursor=%d, want 1,1", m.Offset(), m.Cursor())
	}
}

func TestKeyMap_ZeroLiteralUsesDefaults(t *testing.T) {
	m := Model{ItemCount: 100, Height: 10}
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	if m.Offset() != 1 {
		t.Fatalf("offset=%d, want 1", m.Offset())
	}
	if len(m.Bindings()) == 0 || m.Bindings()[0].Desc == "" {
		t.Fatalf("Bindings() = %+v", m.Bindings())
	}
}

func TestUpdate_EndMovesCursorToLastAndShowsIt(t *testing.T) {
	m := New(1000, 10, func(i int) string { return "" })
	m, _ = m.Update(tui.Key{Type: tui.KeyEnd})
	if m.Cursor() != 999 || m.Offset() > 999 || m.Offset()+10 <= 999 {
		t.Fatalf("cursor=%d offset=%d", m.Cursor(), m.Offset())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyHome})
	if m.Cursor() != 0 || m.Offset() != 0 {
		t.Fatalf("home: cursor=%d offset=%d", m.Cursor(), m.Offset())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyPgDown})
	if m.Cursor() != 10 || m.Offset() != 1 {
		t.Fatalf("pgdown: cursor=%d offset=%d", m.Cursor(), m.Offset())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyPgUp})
	if m.Cursor() != 0 {
		t.Fatalf("pgup: cursor=%d", m.Cursor())
	}
}

func TestMouse_ClickSelectsRowAndWheelScrolls(t *testing.T) {
	m := New(100, 10, func(i int) string { return "" })
	m.Bounds = hittest.Rect{X: 5, Y: 2, W: 20, H: 10}
	click := tui.MouseEvent{X: 6, Y: 5, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
	off, _ := m.Update(click)
	if off.Cursor() != 0 {
		t.Fatal("mouse off must ignore events")
	}
	m.Mouse = true
	m, _ = m.Update(click)
	if m.Cursor() != 3 {
		t.Fatalf("cursor=%d, want 3", m.Cursor())
	}
	wheel := tui.MouseEvent{X: 6, Y: 5, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress}
	m, _ = m.Update(wheel)
	if m.Offset() != 3 {
		t.Fatalf("offset=%d, want default step 3", m.Offset())
	}
	m.WheelStep = 5
	m, _ = m.Update(wheel)
	if m.Offset() != 8 {
		t.Fatalf("offset=%d, want 8", m.Offset())
	}
	out := wheel
	out.X = 0
	m, _ = m.Update(out)
	if m.Offset() != 8 {
		t.Fatalf("outside event changed offset to %d", m.Offset())
	}
	m, _ = m.Update(tui.MouseEvent{X: 6, Y: 4, Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.Offset() != 3 {
		t.Fatalf("offset=%d, want 3", m.Offset())
	}
}
