package taginput

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
)

func TestBindingsListsTagActionsAndInputEditing(t *testing.T) {
	bs := New().Bindings()
	if len(bs) < 3 {
		t.Fatalf("Bindings() has %d entries, want tag actions plus input editing", len(bs))
	}
	if bs[0].Desc != "add tag" || bs[1].Desc != "remove last tag" {
		t.Errorf("first bindings = %q, %q", bs[0].Desc, bs[1].Desc)
	}
	for _, b := range bs {
		if b.Desc == "" || len(b.Keys) == 0 {
			t.Errorf("binding %+v lacks Desc or Keys", b)
		}
	}
}

func TestZeroKeyMapFallsBackToDefaults(t *testing.T) {
	m := Model{}
	m.Input.Focus()
	typeString(&m, "x")
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	if len(m.Tags) != 1 {
		t.Fatalf("Tags = %v, want one tag with zero KeyMap", m.Tags)
	}
}

func TestRebindCommitChangesBehaviour(t *testing.T) {
	m := focused()
	m.KeyMap.Commit = keymap.NewBinding("add tag", "ctrl+j")
	typeString(&m, "x")
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	if len(m.Tags) != 0 {
		t.Fatalf("old key still commits: Tags = %v", m.Tags)
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyCtrl, Code: 'j'})
	if len(m.Tags) != 1 || m.Tags[0] != "x" {
		t.Fatalf("Tags = %v, want [x]", m.Tags)
	}
}
