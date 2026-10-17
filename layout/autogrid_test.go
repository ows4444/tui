package layout

import (
	"strings"
	"testing"
)

func cards(n int) []Node {
	out := make([]Node, n)
	for i := range out {
		out[i] = Block(string(rune('a' + i)))
	}
	return out
}

func TestAutoGridColumnsFollowTheWidth(t *testing.T) {
	for _, tc := range []struct {
		width, cells, wantCols int
	}{
		{45, 8, 2}, {65, 8, 3}, {19, 8, 1}, {20, 8, 1}, {41, 8, 2}, {200, 8, 8},
		{200, 3, 3}, // never more columns than cells
		{1, 4, 1}, {0, 4, 0},
	} {
		a := AutoGrid(20, 1, cards(tc.cells)...).(autoGridNode)
		if got := a.columns(tc.width); got != tc.wantCols && !(tc.width == 0 && got == 1) {
			t.Errorf("width %d, %d cells: %d columns, want %d", tc.width, tc.cells, got, tc.wantCols)
		}
	}
}

func TestAutoGridSharesTheWidthEqually(t *testing.T) {
	root := AutoGrid(20, 1, cards(5)...)
	got := root.Render(Size{W: 45, H: 3})
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("%d lines: %q", len(lines), got)
	}
	for _, l := range lines {
		if len([]rune(l)) != 45 {
			t.Errorf("line %q is %d wide, want 45", l, len([]rune(l)))
		}
	}
	// Two columns of 22 and a gap of 1: the second column starts at x=23.
	if idx := strings.Index(lines[0], "b"); idx != 23 {
		t.Errorf("second cell at x=%d, want 23: %q", idx, lines[0])
	}
	// Three rows of cells (5 cells, 2 columns) with one gap row between: a, c, e on rows 0, 2, 4.
	tall := root.Render(Size{W: 45, H: 5})
	ls := strings.Split(tall, "\n")
	if !strings.HasPrefix(ls[0], "a") || !strings.HasPrefix(ls[2], "c") || !strings.HasPrefix(ls[4], "e") {
		t.Errorf("rows:\n%s", tall)
	}
}

func TestAutoGridRemainderGoesToTheFirstColumns(t *testing.T) {
	root := AutoGrid(10, 1, cards(3)...)
	line := strings.Split(root.Render(Size{W: 33, H: 1}), "\n")[0] // 3 columns: 33-2=31 -> 11, 10, 10
	if strings.Index(line, "b") != 12 || strings.Index(line, "c") != 23 {
		t.Errorf("columns at b=%d c=%d, want 12 and 23: %q", strings.Index(line, "b"), strings.Index(line, "c"), line)
	}
}

func TestAutoGridUnboundedIsOneRow(t *testing.T) {
	root := AutoGrid(5, 1, cards(4)...)
	sz := root.Measure(Constraints{MaxW: Unbounded, MaxH: Unbounded})
	if sz.H != 1 || sz.W != 4*5+3 {
		t.Errorf("Measure = %+v, want 23x1", sz)
	}
}

func TestAutoGridMeasureMatchesRender(t *testing.T) {
	root := AutoGridGaps(12, 2, 1, cards(7)...)
	for _, w := range []int{10, 25, 40, 80} {
		sz := root.Measure(Constraints{MaxW: w, MaxH: Unbounded})
		if sz.W != w && sz.W != 0 {
			t.Errorf("width %d: Measure W = %d, want it to fill the width", w, sz.W)
		}
		out := root.Render(sz)
		if rows := len(strings.Split(out, "\n")); rows != sz.H {
			t.Errorf("width %d: Render has %d rows, Measure says %d", w, rows, sz.H)
		}
	}
}

func TestAutoGridEdges(t *testing.T) {
	if got := AutoGrid(10, 1).Render(Size{W: 20, H: 3}); strings.TrimSpace(got) != "" {
		t.Errorf("no cells drew %q", got)
	}
	if got := AutoGrid(10, 1, cards(2)...).Render(Size{W: 0, H: 3}); got != "" {
		t.Errorf("zero width drew %q", got)
	}
	if sz := AutoGrid(10, 1, cards(2)...).Measure(Constraints{MaxW: 0, MaxH: 5}); sz.W != 0 {
		t.Errorf("Measure at width 0 = %+v", sz)
	}
	// A minWidth under 1 counts as 1 and a negative gap as 0: no panic, all cells fit.
	if got := AutoGrid(0, -4, cards(3)...).Render(Size{W: 3, H: 1}); got != "abc" {
		t.Errorf("degenerate arguments drew %q, want abc", got)
	}
	// The cells slice is copied.
	c := cards(2)
	g := AutoGrid(1, 0, c...)
	c[0] = Block("Z")
	if got := g.Render(Size{W: 2, H: 1}); got != "ab" {
		t.Errorf("a change to the caller's slice reached the grid: %q", got)
	}
}

func TestAutoGridReportsItsCellsToRects(t *testing.T) {
	root := AutoGrid(20, 1, cards(4)...)
	var leaves []Rect
	for _, p := range Rects(root, Size{W: 45, H: 3}) {
		if p.Rect.W > 0 && p.Rect.W < 45 {
			leaves = append(leaves, p.Rect)
		}
	}
	if len(leaves) < 4 {
		t.Errorf("Rects found %d cell rectangles, want 4: %+v", len(leaves), leaves)
	}
}
