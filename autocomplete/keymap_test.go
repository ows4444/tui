package autocomplete

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
)

func TestBindingsListsDropdownAndInputActions(t *testing.T) {
	bs := New("apple").Bindings()
	if len(bs) < 5 {
		t.Fatalf("Bindings() has %d entries, want dropdown plus input editing", len(bs))
	}
	for _, b := range bs {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("binding %+v lacks Desc or Keys", b)
		}
	}
}

func TestZeroKeyMapFallsBackToDefaults(t *testing.T) {
	m := Model{Suggestions: []string{"apple"}}
	m.Input.Focus()
	typeString(&m, "a")
	m, cmd := m.Update(key(tui.KeyTab))
	if cmd == nil || m.Input.Value() != "apple" {
		t.Fatalf("Tab did not accept with zero KeyMap: %q", m.Input.Value())
	}
}

func TestRebindAcceptChangesBehaviour(t *testing.T) {
	m := New("apple")
	m.KeyMap.Accept = keymap.NewBinding("accept", "ctrl+y")
	typeString(&m, "a")
	m, _ = m.Update(key(tui.KeyTab))
	if m.Input.Value() != "a" {
		t.Fatalf("old key still accepts: %q", m.Input.Value())
	}
	m, cmd := m.Update(tui.Key{Type: tui.KeyCtrl, Code: 'y'})
	if cmd == nil || m.Input.Value() != "apple" {
		t.Fatalf("rebound key did not accept: %q", m.Input.Value())
	}
}
