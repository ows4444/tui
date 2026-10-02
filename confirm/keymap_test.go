package confirm

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/keymap"
)

func TestBindingsListsActiveBindingsWithDesc(t *testing.T) {
	bs := New("ok?").Bindings()
	if len(bs) != 4 {
		t.Fatalf("Bindings() has %d entries, want 4", len(bs))
	}
	for _, b := range bs {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("binding %+v lacks Desc or Keys", b)
		}
	}
}

func TestZeroKeyMapFallsBackToDefaults(t *testing.T) {
	m := Model{yes: true}
	_, cmd := m.Update(key(tui.KeyEnter))
	if cmd == nil {
		t.Fatal("Enter did not confirm with zero KeyMap")
	}
}

func TestShiftedYStillConfirms(t *testing.T) {
	_, cmd := New("ok?").Update(tui.Key{Type: tui.KeyRunes, Text: "Y", Mod: input.ModShift})
	if cmd == nil {
		t.Fatal("shift+Y did not confirm")
	}
}

func TestRebindYesChangesBehaviour(t *testing.T) {
	m := New("ok?")
	m.KeyMap.Yes = keymap.NewBinding("yes", "a")
	if _, cmd := m.Update(rk('y')); cmd != nil {
		t.Fatal("old key still confirms")
	}
	_, cmd := m.Update(rk('a'))
	if cmd == nil {
		t.Fatal("rebound key did not confirm")
	}
	if got := cmd().(ConfirmedMsg); !got.Yes {
		t.Fatalf("ConfirmedMsg = %+v, want Yes", got)
	}
}
