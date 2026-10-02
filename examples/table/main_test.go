package main

import (
	"os"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

// The example's Model is built around datatable.Model as its primary
// content, seeded with sample data of at least 3 rows and 2 columns.
func TestModelUsesDataTableAsPrimaryContent(t *testing.T) {
	m := initialModel()

	if len(headers) < 2 {
		t.Fatalf("sample data needs at least 2 columns, got %d", len(headers))
	}
	if len(rows) < 3 {
		t.Fatalf("sample data needs at least 3 rows, got %d", len(rows))
	}
	for i, row := range rows {
		if len(row) != len(headers) {
			t.Fatalf("row %d has %d cells, want %d", i, len(row), len(headers))
		}
	}

	out := ansi.StripANSI(m.table.View())
	for _, want := range []string{headers[0], headers[1], rows[0][0], rows[1][0]} {
		if !strings.Contains(out, want) {
			t.Errorf("table View() missing %q\n%s", want, out)
		}
	}
}

// Up/Down forwarded to the example's Update move the datatable.Model's
// cursor, and the rendered View reflects the new cursor row.
func TestUpDownMovesCursorAndUpdatesView(t *testing.T) {
	m := initialModel()

	if got := m.table.Cursor(); got != 0 {
		t.Fatalf("initial cursor = %d, want 0", got)
	}

	next, cmd := m.Update(tui.Key{Type: tui.KeyDown})
	m = next.(model)
	if cmd != nil {
		t.Fatal("Down should not produce a Cmd")
	}
	if got := m.table.Cursor(); got != 1 {
		t.Fatalf("cursor after Down = %d, want 1", got)
	}

	next, _ = m.Update(tui.Key{Type: tui.KeyDown})
	m = next.(model)
	if got := m.table.Cursor(); got != 2 {
		t.Fatalf("cursor after second Down = %d, want 2", got)
	}

	out := ansi.StripANSI(m.View())
	if !strings.Contains(out, rows[2][0]) {
		t.Errorf("View() should reflect cursor on row 2 (%q)\n%s", rows[2][0], out)
	}

	next, _ = m.Update(tui.Key{Type: tui.KeyUp})
	m = next.(model)
	if got := m.table.Cursor(); got != 1 {
		t.Fatalf("cursor after Up = %d, want 1", got)
	}
}

// Enter emits datatable.Model.SelectedMsg via the returned Cmd; the
// example's Update captures it and the View shows which row was
// selected.
func TestEnterSelectsRowAndViewShowsSelection(t *testing.T) {
	m := initialModel()

	before := ansi.StripANSI(m.View())
	if !strings.Contains(before, "none yet") {
		t.Errorf("View() before selection should indicate nothing selected\n%s", before)
	}

	next, _ := m.Update(tui.Key{Type: tui.KeyDown})
	m = next.(model)

	next, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	m = next.(model)
	if cmd == nil {
		t.Fatal("Enter should return a Cmd delivering SelectedMsg")
	}

	msg := cmd()
	next, _ = m.Update(msg)
	m = next.(model)

	if m.selected == nil {
		t.Fatal("Update should capture SelectedMsg into m.selected")
	}
	if m.selected.Row != 1 {
		t.Errorf("selected.Row = %d, want 1", m.selected.Row)
	}

	out := ansi.StripANSI(m.View())
	if !strings.Contains(out, "Selected:") || !strings.Contains(out, rows[1][0]) {
		t.Errorf("View() should show the selected row %q\n%s", rows[1][0], out)
	}
}

// main.go documents that this example exists specifically to showcase
// datatable.Model's interactive selection, distinct from examples/dashboard
// where a table appears only incidentally.
func TestDocCommentExplainsPurpose(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)

	end := strings.Index(text, "package main")
	if end < 0 {
		t.Fatal("main.go missing package clause")
	}
	doc := text[:end]

	for _, want := range []string{"datatable.Model", "dashboard", "incidentally"} {
		if !strings.Contains(doc, want) {
			t.Errorf("doc comment missing %q:\n%s", want, doc)
		}
	}
}
