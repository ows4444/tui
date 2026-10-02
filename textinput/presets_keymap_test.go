package textinput_test

import (
	"github.com/ows4444/tui/textinput"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
)

func TestPresetKeymapBindingsComeFromTextinput(t *testing.T) {
	if len(textinput.NewSearch().Bindings()) == 0 {
		t.Fatal("Bindings() is empty")
	}
}

func TestPresetKeymapRebindHomeChangesBehaviour(t *testing.T) {
	m := textinput.NewSearch()
	m.Focus()
	m.KeyMap.Home = keymap.NewBinding("start", "ctrl+b")
	for _, r := range "abc" {
		m, _ = m.Update(rk(r))
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyCtrl, Code: 'a'})
	if m.Cursor() != 3 {
		t.Fatalf("old key still bound: Cursor() = %d", m.Cursor())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyCtrl, Code: 'b'})
	if m.Cursor() != 0 {
		t.Fatalf("rebound key: Cursor() = %d, want 0", m.Cursor())
	}
}
