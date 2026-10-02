package viewport

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
)

func tallViewport() Model {
	m := New(10, 3)
	m.SetContent(numberedLines(20))
	return m
}

func TestBindingsListsActiveBindingsWithDesc(t *testing.T) {
	bs := New(10, 3).Bindings()
	if len(bs) != 6 {
		t.Fatalf("Bindings() has %d entries, want 6", len(bs))
	}
	for _, b := range bs {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("binding %+v lacks Desc or Keys", b)
		}
	}
}

func TestZeroKeyMapFallsBackToDefaults(t *testing.T) {
	m := Model{Width: 10, Height: 3}
	m.SetContent(numberedLines(20))
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	if m.yOffset != 1 {
		t.Fatalf("yOffset = %d, want 1 with zero KeyMap", m.yOffset)
	}
}

func TestDefaultKeyNamesMatchKeyString(t *testing.T) {
	km := DefaultKeyMap()
	for _, c := range []struct {
		b keymap.Binding
		k tui.KeyType
	}{
		{km.PageUp, tui.KeyPgUp}, {km.PageDown, tui.KeyPgDown}, {km.Top, tui.KeyHome}, {km.Bottom, tui.KeyEnd},
	} {
		if !keymap.Matches(tui.Key{Type: c.k}, c.b) {
			t.Errorf("default %q does not match its key", c.b.Desc)
		}
	}
}

func TestRebindDownChangesBehaviour(t *testing.T) {
	m := tallViewport()
	m.KeyMap.Down = keymap.NewBinding("down", "j")
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	if m.yOffset != 0 {
		t.Fatalf("old key still scrolls: yOffset = %d", m.yOffset)
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "j"})
	if m.yOffset != 1 {
		t.Fatalf("rebound key: yOffset = %d, want 1", m.yOffset)
	}
}
