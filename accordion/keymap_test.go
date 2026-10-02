package accordion

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
)

func TestBindingsListsActiveBindingsWithDesc(t *testing.T) {
	bs := New(sections()...).Bindings()
	if len(bs) != 3 {
		t.Fatalf("Bindings() has %d entries, want 3", len(bs))
	}
	for _, b := range bs {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("binding %+v lacks Desc or Keys", b)
		}
	}
}

func TestDefaultToggleMatchesEnterAndSpace(t *testing.T) {
	km := DefaultKeyMap()
	if !keymap.Matches(key(tui.KeyEnter), km.Toggle) || !keymap.Matches(key(tui.KeySpace), km.Toggle) {
		t.Fatal("default Toggle does not match Enter and Space")
	}
}

func TestZeroKeyMapFallsBackToDefaults(t *testing.T) {
	m := Model{Sections: sections()}
	m, _ = m.Update(key(tui.KeyDown))
	if m.Cursor() != 1 {
		t.Fatalf("Cursor() = %d, want 1 with zero KeyMap", m.Cursor())
	}
}

func TestRebindDownChangesBehaviour(t *testing.T) {
	m := New(sections()...)
	m.KeyMap.Down = keymap.NewBinding("next", "j")
	m, _ = m.Update(key(tui.KeyDown))
	if m.Cursor() != 0 {
		t.Fatalf("old key still moves: Cursor() = %d", m.Cursor())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "j"})
	if m.Cursor() != 1 {
		t.Fatalf("rebound key: Cursor() = %d, want 1", m.Cursor())
	}
}
