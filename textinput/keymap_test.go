package textinput

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/keymap"
)

func TestBindingsListsActiveBindingsWithDesc(t *testing.T) {
	bs := New().Bindings()
	if len(bs) == 0 {
		t.Fatal("Bindings() is empty")
	}
	for _, b := range bs {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("binding %+v lacks Desc or Keys", b)
		}
	}
}

func TestDefaultKeyNamesMatchKeyString(t *testing.T) {
	km := DefaultKeyMap()
	for _, c := range []struct {
		b keymap.Binding
		k tui.Key
	}{
		{km.Left, tui.Key{Type: tui.KeyLeft}},
		{km.Right, tui.Key{Type: tui.KeyRight}},
		{km.WordLeft, tui.Key{Type: tui.KeyLeft, Mod: input.ModCtrl}},
		{km.WordRight, tui.Key{Type: tui.KeyRight, Mod: input.ModAlt}},
		{km.Home, ck('a')},
		{km.End, ck('e')},
		{km.DeleteToStart, ck('u')},
		{km.DeleteToEnd, ck('k')},
		{km.DeleteWordBack, ck('w')},
		{km.DeleteBack, tui.Key{Type: tui.KeyBackspace}},
		{km.DeleteForward, tui.Key{Type: tui.KeyDelete}},
	} {
		if !keymap.Matches(c.k, c.b) {
			t.Errorf("default %q does not match %q", c.b.Desc, c.k.String())
		}
	}
}

func TestZeroKeyMapFallsBackToDefaults(t *testing.T) {
	m := Model{}
	m.Focus()
	typeString(t, &m, "ab")
	m, _ = m.Update(tui.Key{Type: tui.KeyLeft})
	if m.Cursor() != 1 {
		t.Fatalf("Cursor() = %d, want 1 with zero KeyMap", m.Cursor())
	}
}

func TestShiftLeftStillMovesLeft(t *testing.T) {
	m := focused()
	typeString(t, &m, "ab")
	m, _ = m.Update(tui.Key{Type: tui.KeyLeft, Mod: input.ModShift})
	if m.Cursor() != 1 {
		t.Fatalf("Cursor() = %d, want 1", m.Cursor())
	}
}

func TestRebindHomeChangesBehaviour(t *testing.T) {
	m := focused()
	m.KeyMap.Home = keymap.NewBinding("start", "ctrl+b")
	typeString(t, &m, "abc")
	m, _ = m.Update(ck('a'))
	if m.Cursor() != 3 {
		t.Fatalf("old key still bound: Cursor() = %d, want 3", m.Cursor())
	}
	m, _ = m.Update(ck('b'))
	if m.Cursor() != 0 {
		t.Fatalf("rebound key: Cursor() = %d, want 0", m.Cursor())
	}
}
