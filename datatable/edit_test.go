package datatable

import (
	"slices"
	"testing"
)

func editPeople() [][]string { return [][]string{{"bob", "2"}, {"ada", "9"}, {"cy", "5"}} }

func TestSetCellChangesACopy(t *testing.T) {
	orig := editPeople()
	m := New([]string{"name", "n"}, orig)
	if !m.SetCell(1, 1, "10") {
		t.Fatal("SetCell(1,1) = false")
	}
	if m.Rows[1][1] != "10" {
		t.Errorf("Rows[1][1] = %q", m.Rows[1][1])
	}
	if orig[1][1] != "9" {
		t.Errorf("the caller's slice was edited: %v", orig)
	}
	for _, bad := range [][2]int{{-1, 0}, {3, 0}, {0, -1}, {0, 2}} {
		if m.SetCell(bad[0], bad[1], "x") {
			t.Errorf("SetCell%v succeeded", bad)
		}
	}
}

func TestSetCellPadsAShortRow(t *testing.T) {
	m := New([]string{"a", "b", "c"}, [][]string{{"x"}})
	if !m.SetCell(0, 2, "z") || !slices.Equal(m.Rows[0], []string{"x", "", "z"}) {
		t.Errorf("row = %q", m.Rows[0])
	}
}

func TestInsertAndRemoveRow(t *testing.T) {
	m := New([]string{"name", "n"}, editPeople())
	m.InsertRow(1, []string{"dee", "7"})
	m.InsertRow(99, []string{"eve", "1"})
	m.InsertRow(-5, []string{"first", "0"})
	var names []string
	for _, r := range m.Rows {
		names = append(names, r[0])
	}
	if !slices.Equal(names, []string{"first", "bob", "dee", "ada", "cy", "eve"}) {
		t.Errorf("rows = %v", names)
	}
	if !m.RemoveRow(0) || m.RemoveRow(10) || m.RemoveRow(-1) {
		t.Error("RemoveRow results wrong")
	}
	if len(m.Rows) != 5 || m.Rows[0][0] != "bob" {
		t.Errorf("rows = %v", m.Rows)
	}
}

func TestEditingKeepsASortAndAFilter(t *testing.T) {
	m := New([]string{"name", "n"}, editPeople())
	m.Filter = true
	m.SortBy(0, false)
	if m.Rows[0][0] != "ada" {
		t.Fatalf("sorted rows = %v", m.Rows)
	}
	m.InsertRow(0, []string{"aaron", "1"})
	if m.Rows[0][0] != "aaron" || m.Rows[1][0] != "ada" {
		t.Errorf("a new row is not in sort order: %v", m.Rows)
	}
	if got := m.SourceRow(0); got != 0 {
		t.Errorf("SourceRow(0) = %d, want the inserted row 0", got)
	}
	m.SetQuery("a")
	before := len(m.Rows)
	m.SetCell(m.SourceRow(0), 0, "zzz") // no longer matches the filter
	if len(m.Rows) >= before {
		t.Errorf("an edited cell that stopped matching is still shown: %v", m.Rows)
	}
	m.ClearSort()
	m.SetQuery("")
	if len(m.Rows) != 4 {
		t.Errorf("rows after clearing = %v", m.Rows)
	}
}

func TestRemoveRowClampsTheCursor(t *testing.T) {
	m := New([]string{"name", "n"}, editPeople())
	m.SetCursor(2)
	m.RemoveRow(2)
	if m.Cursor() != 1 {
		t.Errorf("cursor = %d, want 1", m.Cursor())
	}
}
