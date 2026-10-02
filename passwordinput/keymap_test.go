package passwordinput

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
)

func TestBindingsDelegatesToTextinput(t *testing.T) {
	if len(New().Bindings()) == 0 {
		t.Fatal("Bindings() is empty")
	}
}

func TestRebindHomeChangesBehaviour(t *testing.T) {
	m := focused()
	m.KeyMap.Home = keymap.NewBinding("start", "ctrl+b")
	typeString(t, &m, "abc")
	m, _ = m.Update(tui.Key{Type: tui.KeyCtrl, Code: 'a'})
	if m.Cursor() != 3 {
		t.Fatalf("old key still bound: Cursor() = %d", m.Cursor())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyCtrl, Code: 'b'})
	if m.Cursor() != 0 {
		t.Fatalf("rebound key: Cursor() = %d, want 0", m.Cursor())
	}
}
