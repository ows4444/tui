package form

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
)

func TestBindingsFollowFocusedField(t *testing.T) {
	m := New(
		Field{Name: "a", Label: "A"},
		Field{Name: "b", Label: "B", Kind: FieldSelect, Options: []string{"x", "y"}},
		Field{Name: "c", Label: "C", Kind: FieldCheckbox},
	)
	m.Focus()
	text := m.Bindings()
	m = tab(m)
	sel := m.Bindings()
	m = tab(m)
	box := m.Bindings()
	if len(text) <= len(sel) || len(sel) != 5 || len(box) != 4 {
		t.Fatalf("binding counts text=%d select=%d checkbox=%d", len(text), len(sel), len(box))
	}
	for _, bs := range [][]keymap.Binding{text, sel, box} {
		for _, b := range bs {
			if b.Desc == "" || len(b.Keys) == 0 {
				t.Errorf("binding %+v lacks Desc or Keys", b)
			}
		}
	}
}

func TestZeroKeyMapFallsBackToDefaults(t *testing.T) {
	m := signup()
	m.KeyMap = KeyMap{}
	m.Focus()
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter did not submit with zero KeyMap")
	}
}

func TestRebindSubmitChangesBehaviour(t *testing.T) {
	m := New(Field{Name: "a", Label: "A", Value: "ok"})
	m.KeyMap.Submit = keymap.NewBinding("submit", "ctrl+s")
	m.Focus()
	if _, cmd := m.Update(tui.Key{Type: tui.KeyEnter}); cmd != nil {
		if _, ok := cmd().(SubmittedMsg); ok {
			t.Fatal("old key still submits")
		}
	}
	_, cmd := m.Update(tui.Key{Type: tui.KeyCtrl, Code: 's'})
	if cmd == nil {
		t.Fatal("rebound key did not submit")
	}
	if _, ok := cmd().(SubmittedMsg); !ok {
		t.Fatalf("Cmd produced %T, want SubmittedMsg", cmd())
	}
}
