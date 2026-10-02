package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestTable(t *testing.T) {
	dt := theme.DarkTheme()
	headers := []string{"Name", "Age"}
	rows := [][]string{
		{"Alice", "30"},
		{"Bob", "5"},
	}

	got := Table(headers, rows, dt)

	headerStyle := ansi.NewStyle().Bold().Foreground(dt.Primary)
	dividerStyle := ansi.NewStyle().Foreground(dt.Muted)

	header := headerStyle.Render("Name ") + "  " + headerStyle.Render("Age")
	divider := dividerStyle.Render(strings.Repeat("─", 10)) // 5 + 2 + 3
	row0 := "Alice" + "  " + "30 "
	row1 := "Bob  " + "  " + "5  "

	want := header + "\n" + divider + "\n" + row0 + "\n" + row1
	if got != want {
		t.Errorf("Table() =\n%q\nwant\n%q", got, want)
	}
}

// TestTableIsTableRowsWithUnstyledRows proves acceptance
// criterion for widgets.Table's half: Table is exactly TableRows with an
// always-unstyled per-row style, the same shared implementation
// datatable.Model.View calls directly to highlight its cursor row — not a
// separate, independently-maintained copy of the column-width/padding
// logic.
func TestTableIsTableRowsWithUnstyledRows(t *testing.T) {
	dt := theme.DarkTheme()
	headers := []string{"Name", "Age", "City"}
	rows := [][]string{
		{"Alice", "30", "Springfield"},
		{"Bob", "5", "NYC"},
	}

	headerStyle := ansi.NewStyle().Bold().Foreground(dt.Primary)
	dividerStyle := ansi.NewStyle().Foreground(dt.Muted)

	want := TableRows(headers, rows, headerStyle, dividerStyle, func(int) ansi.Style { return ansi.Style{} })
	got := Table(headers, rows, dt)

	if got != want {
		t.Errorf("Table() diverged from TableRows(): Table() =\n%q\nTableRows() =\n%q", got, want)
	}
}

func TestTableColumnWidthAwareOfStyledCells(t *testing.T) {
	dt := theme.DarkTheme()
	styled := ansi.NewStyle().Bold().Render("x") // visible width 1, byte length much more
	got := Table([]string{"A"}, [][]string{{styled}}, dt)
	lines := strings.Split(got, "\n")
	if ansi.Width(lines[0]) != 1 {
		t.Fatalf("header width = %d, want 1 (column should size to the styled cell's visible width, not its byte length)", ansi.Width(lines[0]))
	}
}

func TestTableEmptyHeaders(t *testing.T) {
	if got := Table(nil, [][]string{{"a"}}, theme.DarkTheme()); got != "" {
		t.Errorf("Table(nil headers) = %q, want empty", got)
	}
}

func TestTableNoRows(t *testing.T) {
	dt := theme.DarkTheme()
	got := Table([]string{"X"}, nil, dt)
	headerStyle := ansi.NewStyle().Bold().Foreground(dt.Primary)
	dividerStyle := ansi.NewStyle().Foreground(dt.Muted)
	want := headerStyle.Render("X") + "\n" + dividerStyle.Render("─")
	if got != want {
		t.Errorf("Table() with no rows = %q, want %q", got, want)
	}
}

func TestTableShortRowPadded(t *testing.T) {
	dt := theme.DarkTheme()
	// A row with fewer cells than headers should just render blanks for
	// the missing ones, not panic or misalign.
	got := Table([]string{"A", "B", "C"}, [][]string{{"1"}}, dt)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3 (header, divider, one row)", len(lines))
	}
	// col0="1" (width 1, no pad) + "  " gap + col1 blank (1 pad space) +
	// "  " gap + col2 blank (1 pad space) = "1" followed by 6 spaces.
	dataRow := ansi.StripANSI(lines[2])
	want := "1" + strings.Repeat(" ", 6)
	if dataRow != want {
		t.Errorf("short row = %q, want %q", dataRow, want)
	}
}
