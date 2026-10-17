package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

var jobs = [][]string{
	{"build", "ok", "12.5"},
	{"deploy", "failed", "3"},
	{"lint", "ok", "100"},
}

var jobHeaders = []string{"Job", "Status", "Time"}

func tcLines(s string) []string { return strings.Split(ansi.StripANSI(s), "\n") }

func TestTableCellsWithNoOptionsIsTable(t *testing.T) {
	th := theme.DarkTheme()
	if got, want := TableCells(jobHeaders, jobs, TableOptions{}, th), Table(jobHeaders, jobs, th); got != want {
		t.Errorf("TableCells with no options differs from Table:\n%q\n%q", got, want)
	}
	asc := theme.DarkTheme().ASCII()
	if got, want := TableCells(jobHeaders, jobs, TableOptions{}, asc), Table(jobHeaders, jobs, asc); got != want {
		t.Errorf("on an ASCII theme:\n%q\n%q", got, want)
	}
	if got := TableCells(nil, jobs, TableOptions{}, th); got != "" {
		t.Errorf("no headers drew %q", got)
	}
}

func TestTableCellsAlignsColumnsAndHeaders(t *testing.T) {
	got := tcLines(TableCells(jobHeaders, jobs, TableOptions{
		Align: []layout.Align{layout.AlignStart, layout.AlignCenter, layout.AlignEnd},
	}, theme.DarkTheme()))
	want := []string{
		"Job     Status  Time",
		"build     ok    12.5",
		"deploy  failed     3",
		"lint      ok     100",
	}
	if got[0] != want[0] || got[2] != want[1] || got[3] != want[2] || got[4] != want[3] {
		t.Errorf("rows:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// Every row is the same width.
	for _, l := range got {
		if ansi.Width(l) != ansi.Width(got[0]) {
			t.Errorf("row %q is %d wide, header is %d", l, ansi.Width(l), ansi.Width(got[0]))
		}
	}
	// A column past the end of Align is flush left.
	left := tcLines(TableCells(jobHeaders, jobs, TableOptions{Align: []layout.Align{layout.AlignEnd}}, theme.DarkTheme()))
	if !strings.HasPrefix(left[2], " build  ok  ") {
		t.Errorf("default alignment: %q", left[2])
	}
}

func TestTableCellsStylesOneCellOverItsRow(t *testing.T) {
	red := ansi.NewStyle().Foreground(ansi.Red)
	dim := ansi.NewStyle().Faint()
	out := TableCells(jobHeaders, jobs, TableOptions{
		RowStyle: func(i int) ansi.Style {
			if i == 1 {
				return dim
			}
			return ansi.Style{}
		},
		CellStyle: func(i, j int) ansi.Style {
			if i == 1 && j == 1 {
				return red
			}
			return ansi.Style{}
		},
	}, theme.DarkTheme())
	lines := strings.Split(out, "\n")
	if !strings.Contains(lines[3], red.Render("failed")) {
		t.Errorf("the failed cell is not red: %q", lines[3])
	}
	if !strings.Contains(lines[3], dim.Render("deploy")) || !strings.Contains(lines[3], dim.Render("3   ")) {
		t.Errorf("the rest of the row lost its row style: %q", lines[3])
	}
	if strings.Contains(lines[2], "\x1b[2m") || strings.Contains(lines[2], red.Render("ok")) {
		t.Errorf("another row was styled: %q", lines[2])
	}
}

func TestTableCellsStyleCoversThePaddedCell(t *testing.T) {
	bold := ansi.NewStyle().Bold()
	out := TableCells(jobHeaders, jobs, TableOptions{
		CellStyle: func(i, j int) ansi.Style {
			if j == 0 && i == 2 {
				return bold
			}
			return ansi.Style{}
		},
	}, theme.DarkTheme())
	if !strings.Contains(out, bold.Render("lint  ")) {
		t.Errorf("the style should cover the padding so a background fills the cell: %q", out)
	}
}

func TestTableCellsToleratesRaggedAndWideRows(t *testing.T) {
	rows := [][]string{{"日本語"}, {"a", "b", "c", "extra"}, {}}
	out := tcLines(TableCells(jobHeaders, rows, TableOptions{Align: []layout.Align{layout.AlignEnd}}, theme.DarkTheme()))
	if len(out) != 5 {
		t.Fatalf("%d lines", len(out))
	}
	for _, l := range out {
		if ansi.Width(l) != ansi.Width(out[0]) {
			t.Errorf("row %q is %d wide, want %d", l, ansi.Width(l), ansi.Width(out[0]))
		}
	}
	if strings.Contains(strings.Join(out, "\n"), "extra") {
		t.Error("a cell past the last column was drawn")
	}
	// A cell with its own styling is measured without it.
	styled := TableCells([]string{"A"}, [][]string{{ansi.NewStyle().Bold().Render("x")}, {"yyy"}}, TableOptions{Align: []layout.Align{layout.AlignEnd}}, theme.DarkTheme())
	if lines := tcLines(styled); lines[2] != "  x" || lines[3] != "yyy" {
		t.Errorf("styled cell: %q", lines)
	}
}

func TestTableCellsCentresOddPaddingLeftHeavy(t *testing.T) {
	got := tcLines(TableCells([]string{"Name", "N"}, [][]string{{"a", "long"}}, TableOptions{
		Align: []layout.Align{layout.AlignCenter, layout.AlignEnd},
	}, theme.DarkTheme()))
	// "a" in 4 columns has 3 spare: one on the left, two on the right (as datatable does).
	// The short header "N" follows its column's alignment over the wider cell.
	if got[0] != "Name     N" || got[2] != " a    long" {
		t.Errorf("rows %q", got)
	}
}
