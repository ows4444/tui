package treeview

import (
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/typeahead"
)

func typed(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func TestTypeAheadJumpsAmongVisibleNodes(t *testing.T) {
	m := New(
		Node{Label: "src", Children: []Node{{Label: "main.go"}, {Label: "mod.go"}}},
		Node{Label: "docs"},
		Node{Label: "Makefile"},
	)
	m.TypeAhead = true
	clock := time.Unix(10, 0)
	typeahead.Now = func() time.Time { return clock }
	defer func() { typeahead.Now = time.Now }()
	m = typed(m, "m")
	if m.Cursor() != 2 { // src is collapsed, so main.go is not a candidate
		t.Fatalf("m -> %d, want 2 (Makefile)", m.Cursor())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyUp})
	m, _ = m.Update(tui.Key{Type: tui.KeyUp})
	m, _ = m.Update(tui.Key{Type: tui.KeyRight}) // expand src
	clock = clock.Add(2 * time.Second)           // a pause: a new word
	m = typed(m, "mo")
	if got := m.flatten()[m.Cursor()].label; got != "mod.go" {
		t.Errorf("mo -> %q", got)
	}
}

func TestTypeAheadOffByDefaultAndFilterWins(t *testing.T) {
	m := New(Node{Label: "alpha"}, Node{Label: "beta"})
	if m = typed(m, "b"); m.Cursor() != 0 {
		t.Error("letters moved the cursor with TypeAhead off")
	}
	m.TypeAhead, m.Filter = true, true
	m = typed(m, "b")
	if m.Query() != "b" {
		t.Errorf("Filter did not take the letter: %q", m.Query())
	}
}
