package picker

import (
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/internal/typeahead"
)

func typeKey(m Model, s string) Model {
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: s})
	return m
}

func TestTypeAheadJumpsToTheNextLabelThatStartsWithTheLetters(t *testing.T) {
	m := NewStrings("Apple", "Avocado", "Banana", "Blueberry", "Cherry")
	m.TypeAhead = true
	clock := time.Unix(10, 0)
	typeahead.Now = func() time.Time { return clock }
	defer func() { typeahead.Now = time.Now }()
	m = typeKey(m, "b")
	if m.Cursor() != 2 {
		t.Fatalf("b -> %d, want 2", m.Cursor())
	}
	m = typeKey(m, "b") // the same letter again goes on
	if m.Cursor() != 3 {
		t.Fatalf("b b -> %d, want 3", m.Cursor())
	}
	clock = clock.Add(2 * time.Second)
	m = typeKey(m, "c")
	if m.Cursor() != 4 {
		t.Fatalf("c -> %d, want 4", m.Cursor())
	}
	clock = clock.Add(2 * time.Second)
	m = typeKey(typeKey(m, "A"), "v") // case is ignored and the word grows in place
	if m.Cursor() != 1 {
		t.Fatalf("Av -> %d, want 1", m.Cursor())
	}
	if it := m.Highlighted(); it.Label != "Avocado" {
		t.Errorf("Highlighted = %v", it)
	}
}

func TestTypeAheadIsOffByDefaultAndKeepsBindings(t *testing.T) {
	m := NewStrings("Apple", "Banana")
	if m = typeKey(m, "b"); m.Cursor() != 0 {
		t.Errorf("letters moved the cursor with TypeAhead off: %d", m.Cursor())
	}
	m.TypeAhead = true
	m.KeyMap = DefaultKeyMap()
	m.KeyMap.Down.Keys = append([]string{"j"}, m.KeyMap.Down.Keys...)
	if m = typeKey(m, "j"); m.Cursor() != 1 {
		t.Errorf("a bound letter (j) did not move down: %d", m.Cursor())
	}
	for _, k := range []tui.Key{
		{Type: tui.KeyRunes, Text: "a", Mod: input.ModCtrl},
		{Type: tui.KeyRunes, Text: "a", Mod: input.ModAlt},
		{Type: tui.KeyEnter},
	} {
		m2, _ := m.Update(k)
		if m2.Cursor() != m.Cursor() && k.Type != tui.KeyEnter {
			t.Errorf("%v moved the cursor", k)
		}
	}
}

func TestFilterWinsOverTypeAhead(t *testing.T) {
	m := NewStrings("Apple", "Banana", "Blueberry")
	m.Filter, m.TypeAhead = true, true
	if m = typeKey(m, "b"); m.Query() != "b" {
		t.Errorf("the filter did not take the letter: query %q", m.Query())
	}
}
