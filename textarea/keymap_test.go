package textarea

import (
	"testing"

	"github.com/ows4444/tui"
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
		{km.Newline, tui.Key{Type: tui.KeyEnter}},
		{km.Up, tui.Key{Type: tui.KeyUp}},
		{km.Down, tui.Key{Type: tui.KeyDown}},
		{km.Home, tui.Key{Type: tui.KeyCtrl, Code: 'a'}},
		{km.DeleteWordBack, tui.Key{Type: tui.KeyCtrl, Code: 'w'}},
	} {
		if !keymap.Matches(c.k, c.b) {
			t.Errorf("default %q does not match %q", c.b.Desc, c.k.String())
		}
	}
}

func TestZeroKeyMapFallsBackToDefaults(t *testing.T) {
	m := Model{}
	m.Focus()
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	if m.Value() != "\n" {
		t.Fatalf("Value() = %q, want a newline with zero KeyMap", m.Value())
	}
}

func TestRebindNewlineChangesBehaviour(t *testing.T) {
	m := focused()
	m.KeyMap.Newline = keymap.NewBinding("new line", "ctrl+j")
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	if m.Value() != "" {
		t.Fatalf("old key still bound: Value() = %q", m.Value())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyCtrl, Code: 'j'})
	if m.Value() != "\n" {
		t.Fatalf("rebound key: Value() = %q, want newline", m.Value())
	}
}
