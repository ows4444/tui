package toolapproval

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
)

func TestBindingsListsActiveBindingsWithDesc(t *testing.T) {
	bs := New("rm", "delete", RiskHigh).Bindings()
	if len(bs) != 3 {
		t.Fatalf("Bindings() has %d entries, want 3", len(bs))
	}
	for _, b := range bs {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("binding %+v lacks Desc or Keys", b)
		}
	}
}

func TestZeroKeyMapFallsBackToDefaults(t *testing.T) {
	m := Model{}
	m, _ = m.Update(key(tui.KeyRight))
	if m.Highlighted() != ChoiceDeny {
		t.Fatalf("Highlighted() = %v, want Deny with zero KeyMap", m.Highlighted())
	}
}

func TestRebindNextChangesBehaviour(t *testing.T) {
	m := New("rm", "delete", RiskLow)
	m.KeyMap.Next = keymap.NewBinding("next", "l")
	m, _ = m.Update(key(tui.KeyRight))
	if m.Highlighted() != ChoiceApprove {
		t.Fatal("old key still moves")
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "l"})
	if m.Highlighted() != ChoiceDeny {
		t.Fatalf("Highlighted() = %v, want Deny after rebound key", m.Highlighted())
	}
}
