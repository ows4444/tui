package virtuallist

import (
	"fmt"
	"testing"

	"github.com/ows4444/tui"
)

func letters(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func TestTypeAheadOnAHugeList(t *testing.T) {
	calls := 0
	m := New(1_000_000, 10, func(i int) string { return fmt.Sprint(i) })
	m.Label = func(i int) string { calls++; return fmt.Sprintf("item %07d", i) }
	m = letters(m, "item 0000")
	if m.Cursor() != 0 {
		t.Fatalf("cursor %d after item 0000", m.Cursor())
	}
	m = letters(m, "5")
	if m.Cursor() != 500 || calls > 1000 {
		t.Errorf("cursor %d after \"item 00005\", %d Label calls: the search must stop at the first match", m.Cursor(), calls)
	}
}

func TestTypeAheadOffWithoutALabelAndBindingsWin(t *testing.T) {
	m := New(5, 3, func(i int) string { return fmt.Sprint(i) })
	if m = letters(m, "x"); m.Cursor() != 0 {
		t.Error("letters moved a list with no Label")
	}
	m.Label = func(i int) string { return []string{"a", "b", "c", "d", "e"}[i] }
	m.KeyMap = DefaultKeyMap()
	m.KeyMap.Down.Keys = append([]string{"d"}, m.KeyMap.Down.Keys...)
	if m = letters(m, "d"); m.Cursor() != 1 {
		t.Errorf("a bound letter (d) was taken for type-ahead: cursor %d", m.Cursor())
	}
}

func TestTypeAheadScrollsTheViewToTheMatch(t *testing.T) {
	m := New(100, 5, func(i int) string { return fmt.Sprint(i) })
	m.Label = func(i int) string { return fmt.Sprintf("n%03d", i) }
	m = letters(m, "n070")
	if m.Cursor() != 70 {
		t.Fatalf("cursor %d", m.Cursor())
	}
	if m.offset > 70 || m.offset+5 <= 70 {
		t.Errorf("offset %d does not show item 70", m.offset)
	}
}
