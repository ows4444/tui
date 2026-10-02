package errorretry

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/keymap"
)

func TestBindingsListsActiveBindingsWithDesc(t *testing.T) {
	m := New("boom", 1)
	bs := m.Bindings()
	if len(bs) != 2 {
		t.Fatalf("Bindings() has %d entries, want 2", len(bs))
	}
	for _, b := range bs {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("binding %+v lacks Desc or Keys", b)
		}
	}
	m, _ = m.Update(key(tui.KeyEnter))
	if got := m.Bindings(); len(got) != 1 || got[0].Desc != "dismiss" {
		t.Fatalf("exhausted Bindings() = %+v, want dismiss only", got)
	}
}

func TestZeroKeyMapFallsBackToDefaults(t *testing.T) {
	m := Model{MaxRetries: 1}
	_, cmd := m.Update(key(tui.KeyEsc))
	if cmd == nil {
		t.Fatal("Esc did not dismiss with zero KeyMap")
	}
}

func TestShiftedRStillRetries(t *testing.T) {
	_, cmd := New("boom", 1).Update(tui.Key{Type: tui.KeyRunes, Text: "R", Mod: input.ModShift})
	if cmd == nil {
		t.Fatal("shift+R did not retry")
	}
}

func TestRebindRetryChangesBehaviour(t *testing.T) {
	m := New("boom", 2)
	m.KeyMap.Retry = keymap.NewBinding("retry", "ctrl+r")
	m, _ = m.Update(key(tui.KeyEnter))
	if m.RetryCount() != 0 {
		t.Fatal("old key still retries")
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyCtrl, Code: 'r'})
	if m.RetryCount() != 1 {
		t.Fatalf("RetryCount() = %d, want 1 after rebound key", m.RetryCount())
	}
}
