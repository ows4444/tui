package datatable

import (
	"testing"

	"github.com/ows4444/tui"
)

func typed(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func staff() Model {
	return New([]string{"Name", "City"}, [][]string{
		{"Ada", "London"}, {"Bob", "Berlin"}, {"Bea", "Paris"}, {"Cy", "Rome"},
	})
}

func TestTypeAheadJumpsByTheFirstColumn(t *testing.T) {
	m := staff()
	m.TypeAhead = true
	if m = typed(m, "b"); m.Cursor() != 1 {
		t.Fatalf("b -> %d, want 1", m.Cursor())
	}
	m.ta.Clear()
	if m = typed(m, "be"); m.Cursor() != 2 {
		t.Errorf("be -> %d, want 2", m.Cursor())
	}
}

func TestTypeAheadUsesTheCurrentColumnInCellMode(t *testing.T) {
	m := staff()
	m.TypeAhead, m.CellMode = true, true
	m, _ = m.Update(tui.Key{Type: tui.KeyRight}) // the City column
	if m = typed(m, "p"); m.Cursor() != 2 {
		t.Errorf("p in the City column -> %d, want 2 (Paris)", m.Cursor())
	}
}

func TestBoundLettersKeepTheirMeaningAndOffMeansOff(t *testing.T) {
	m := staff()
	if m = typed(m, "c"); m.Cursor() != 0 {
		t.Error("letters moved the cursor with TypeAhead off")
	}
	m.TypeAhead = true
	m = typed(m, "s") // "s" sorts by default, so it is not a type-ahead letter
	if m.Cursor() != 0 {
		t.Errorf("s moved the cursor: %d", m.Cursor())
	}
	if m.SourceRow(0) == 0 && m.rowAt(0)[0] != "Ada" {
		t.Errorf("s did not sort: first row %v", m.rowAt(0))
	}
}

type threeRows struct{}

func (threeRows) Rows() int          { return 3 }
func (threeRows) Row(i int) []string { return []string{[]string{"ann", "bob", "cy"}[i]} }

func TestTypeAheadSkipsATableWithASource(t *testing.T) {
	m := NewSource([]string{"Name"}, threeRows{})
	m.TypeAhead = true
	if m = typed(m, "b"); m.Cursor() != 0 {
		t.Errorf("type-ahead ran on a Source table: cursor %d", m.Cursor())
	}
}
