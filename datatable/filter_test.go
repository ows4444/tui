package datatable

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
)

func people() Model {
	m := New([]string{"Name", "Age", "City"}, [][]string{
		{"Carol", "41", "Lahore"},
		{"alice", "30", "Tokyo"},
		{"Bob", "5", "Lahore"},
		{"Dave", "100", "Osaka"},
	})
	m.Filter = true
	return m
}

func names(m Model) string {
	var n []string
	for _, r := range m.Rows {
		n = append(n, r[0])
	}
	return strings.Join(n, ",")
}

func typeQuery(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r})
	}
	return m
}

func openFilter(m Model) Model {
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "/", Code: '/'})
	return m
}

func TestSortByAndClearSortRestoreTheGivenOrder(t *testing.T) {
	m := people()
	m.SortBy(1, false) // Age ascending: numbers, not strings
	if got := names(m); got != "Bob,alice,Carol,Dave" {
		t.Fatalf("ascending = %s", got)
	}
	m.SortBy(1, true)
	if got := names(m); got != "Dave,Carol,alice,Bob" {
		t.Fatalf("descending = %s", got)
	}
	if c, desc, ok := m.Sorted(); c != 1 || !desc || !ok {
		t.Errorf("Sorted = %d %v %v", c, desc, ok)
	}
	m.ClearSort()
	if got := names(m); got != "Carol,alice,Bob,Dave" {
		t.Errorf("after ClearSort = %s", got)
	}
	if _, _, ok := m.Sorted(); ok {
		t.Error("still sorted after ClearSort")
	}
	m.SortBy(9, false) // outside the headers
	if _, _, ok := m.Sorted(); ok {
		t.Error("SortBy outside the headers sorted")
	}
}

func TestSortKeyAndUnsortKey(t *testing.T) {
	m := people()
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "s", Code: 's'})
	// Text compares byte by byte, as it always has: capitals first.
	if got := names(m); got != "Bob,Carol,Dave,alice" {
		t.Fatalf("after s = %s", got)
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "S", Code: 'S'})
	if got := names(m); got != "Carol,alice,Bob,Dave" {
		t.Errorf("after S = %s", got)
	}
}

func TestCursorFollowsItsRowThroughASort(t *testing.T) {
	m := people()
	m.SetCursor(2) // Bob
	m.SortBy(1, false)
	if m.Rows[m.Cursor()][0] != "Bob" {
		t.Errorf("cursor on %q after sort, want Bob", m.Rows[m.Cursor()][0])
	}
	m.ClearSort()
	if m.Rows[m.Cursor()][0] != "Bob" {
		t.Errorf("cursor on %q after ClearSort, want Bob", m.Rows[m.Cursor()][0])
	}
}

func TestSetRowsKeepsTheSortApplied(t *testing.T) {
	m := people()
	m.SortBy(0, false)
	m.SetRows([][]string{{"Zed", "1", "x"}, {"Amy", "2", "y"}})
	if got := names(m); got != "Amy,Zed" {
		t.Errorf("after SetRows = %s, want them sorted", got)
	}
	m.ClearSort()
	if got := names(m); got != "Zed,Amy" {
		t.Errorf("after ClearSort = %s, want the given order", got)
	}
}

func TestFilterOffIgnoresTheKeys(t *testing.T) {
	m := people()
	m.Filter = false
	m = openFilter(m)
	if m.Filtering() || m.Query() != "" {
		t.Error("filter opened with Filter off")
	}
	m.SetQuery("bob")
	if m.Query() != "" || len(m.Rows) != 4 {
		t.Error("SetQuery worked with Filter off")
	}
}

func TestFilterNarrowsAcrossCells(t *testing.T) {
	m := people()
	m = openFilter(m)
	if !m.Filtering() {
		t.Fatal("/ did not open the filter line")
	}
	m = typeQuery(m, "lah") // matches the city
	if got := names(m); got != "Carol,Bob" {
		t.Fatalf("rows = %s, want the two in Lahore in their order", got)
	}
	view := ansi.StripANSI(m.View())
	if !strings.HasPrefix(view, "/ lah_") || !strings.Contains(view, "2 of 4") {
		t.Errorf("view:\n%s", view)
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter}) // close, keep
	if m.Filtering() || m.Query() != "lah" || len(m.Rows) != 2 {
		t.Errorf("Enter: filtering %v query %q rows %d", m.Filtering(), m.Query(), len(m.Rows))
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyEsc}) // not editing: clears
	if m.Query() != "" || len(m.Rows) != 4 {
		t.Errorf("Esc: query %q rows %d", m.Query(), len(m.Rows))
	}
}

func TestTypingDoesNotTriggerBindings(t *testing.T) {
	m := openFilter(people())
	m = typeQuery(m, "s") // "s" is the sort key outside the filter line
	if _, _, ok := m.Sorted(); ok {
		t.Error("typing s in the filter line sorted the table")
	}
	if m.Query() != "s" {
		t.Errorf("query = %q", m.Query())
	}
}

func TestFilterEscWhileEditingClearsAndCloses(t *testing.T) {
	m := typeQuery(openFilter(people()), "zzz")
	if len(m.Rows) != 0 {
		t.Fatalf("rows = %d, want none", len(m.Rows))
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyBackspace})
	m, _ = m.Update(tui.Key{Type: tui.KeyBackspace})
	m, _ = m.Update(tui.Key{Type: tui.KeyBackspace})
	if len(m.Rows) != 4 || m.Query() != "" {
		t.Errorf("after Backspace: %d rows, query %q", len(m.Rows), m.Query())
	}
	m = typeQuery(m, "x")
	m, _ = m.Update(tui.Key{Type: tui.KeyEsc})
	if m.Filtering() || m.Query() != "" {
		t.Error("Esc while editing left the filter")
	}
}

func TestSelectedMsgCarriesTheSourceRow(t *testing.T) {
	m := people()
	m.SortBy(1, false)                              // Bob, alice, Carol, Dave
	m.SetQuery("o")                                 // Bob, Carol, Tokyo(alice), Osaka(Dave)... narrow instead:
	m.SetQuery("lah")                               // Bob, Carol
	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter}) // cursor on Bob
	sel := cmd().(SelectedMsg)
	if sel.Cells[0] != "Bob" || sel.Row != 0 || sel.SourceRow != 2 {
		t.Errorf("SelectedMsg = %+v, want Bob row 0 source 2", sel)
	}
	if got := m.SourceRow(1); got != 0 { // Carol
		t.Errorf("SourceRow(1) = %d, want 0", got)
	}
	if m.SourceRow(9) != -1 {
		t.Error("SourceRow out of range")
	}
}

func TestFilterLineShiftsMouseRows(t *testing.T) {
	m := typeQuery(openFilter(people()), "lah")
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 40, H: 8}
	// Line 0 filter, 1 header, 2 divider, 3 first row, 4 second row.
	m, _ = m.Update(tui.MouseEvent{X: 4, Y: 4, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Rows[m.Cursor()][0] != "Bob" {
		t.Errorf("click on line 4 -> %q, want Bob", m.Rows[m.Cursor()][0])
	}
}

func TestFilterLayoutNodeAndDrawCells(t *testing.T) {
	m := typeQuery(openFilter(people()), "lah")
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	if sz := m.LayoutNode().Measure(layout.Unconstrained()); sz.H != 5 { // line, header, divider, 2 rows
		t.Errorf("Measure H = %d, want 5", sz.H)
	}
	lines := strings.Split(ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 40, H: 5})), "\n")
	if len(lines) != 5 || !strings.HasPrefix(lines[0], "/ lah") || !strings.Contains(lines[3], "Carol") {
		t.Errorf("Render:\n%s", strings.Join(lines, "\n"))
	}
	sameScreen(t, m, 40, 5)
}

func TestFilterLinearizeSpeaksTheShownRows(t *testing.T) {
	m := people()
	m.SetQuery("tokyo")
	got := m.Linearize()
	if !strings.Contains(got, "Row 1 of 1: Name: alice") || strings.Contains(got, "Carol") {
		t.Errorf("Linearize = %q", got)
	}
}

func TestFilterLargeTable(t *testing.T) {
	rows := make([][]string, 1000)
	for i := range rows {
		rows[i] = []string{fmt.Sprint("n", i), fmt.Sprint(1000 - i)}
	}
	m := New([]string{"N", "V"}, rows)
	m.Filter = true
	m.SetQuery("n99")
	if len(m.Rows) == 0 || len(m.Rows) >= 1000 {
		t.Errorf("rows = %d", len(m.Rows))
	}
}
