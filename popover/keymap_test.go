package popover

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
)

func TestBindingsListsDismissWithDesc(t *testing.T) {
	bs := New("c", 0, 0).Bindings()
	if len(bs) != 1 || bs[0].Desc == "" || len(bs[0].Keys) == 0 {
		t.Fatalf("Bindings() = %+v", bs)
	}
}

func TestBindingsEmptyWhenClosed(t *testing.T) {
	m := New("c", 0, 0)
	m.Hide()
	if len(m.Bindings()) != 0 {
		t.Fatalf("closed popover Bindings() = %+v, want none", m.Bindings())
	}
}

func TestZeroKeyMapFallsBackToDefaults(t *testing.T) {
	m := Model{}
	m.Show()
	m, _ = m.Update(key(tui.KeyEsc))
	if m.Open() {
		t.Fatal("Esc did not dismiss with zero KeyMap")
	}
}

func TestRebindDismissChangesBehaviour(t *testing.T) {
	m := New("c", 0, 0)
	m.KeyMap.Dismiss = keymap.NewBinding("dismiss", "q")
	m, _ = m.Update(key(tui.KeyEsc))
	if !m.Open() {
		t.Fatal("old key still dismisses")
	}
	m, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "q"})
	if m.Open() || cmd == nil {
		t.Fatal("rebound key did not dismiss")
	}
}
