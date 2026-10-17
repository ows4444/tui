package layout

import (
	"sort"
	"strings"

	"github.com/ows4444/tui/ansi"
)

// spanNode marks a grid cell that covers more than one column or row. Outside
// a grid it is just its child.
type spanNode struct {
	Node
	cols, rows int
}

// Span makes n cover cols columns and rows rows when it is a cell of GridNode
// or GridNodeGaps; a value under 1 counts as 1. Cells fill the grid row by row
// as before, skipping the places a span already covers, so
//
//	GridNode([]Track{{}, {}}, 1, Span(wide, 2, 1), left, right)
//
// puts wide across the top and left and right under it. A span wider than the
// grid is cut to the grid's columns, and one that does not fit in what is left
// of a row starts the next row. A spanning cell is stretched over the columns
// and rows it covers, and the gaps between them; if it is wider or taller than
// they are, the columns or rows it covers grow to hold it (auto-sized columns
// before fixed ones, which never grow). Outside a grid, Span is n.
func Span(n Node, cols, rows int) Node {
	if sp, ok := n.(spanNode); ok {
		n = sp.Node
	}
	return spanNode{Node: n, cols: max(cols, 1), rows: max(rows, 1)}
}

func (s spanNode) DrawCells(dst CellSurface, r Rect) { drawNode(s.Node, dst, r) }

// cellSpan is how many columns and rows grid cell i covers.
type cellSpan struct{ cols, rows int }

// withSpans lifts Span wrappers out of cells into the grid's spans table.
func (g gridNode) withSpans() gridNode {
	any := false
	for _, c := range g.cells {
		if _, ok := c.(spanNode); ok {
			any = true
			break
		}
	}
	if !any {
		return g
	}
	cells := make([]Node, len(g.cells))
	spans := make([]cellSpan, len(g.cells))
	for i, c := range g.cells {
		cells[i], spans[i] = c, cellSpan{1, 1}
		if sp, ok := c.(spanNode); ok {
			cells[i], spans[i] = sp.Node, cellSpan{sp.cols, sp.rows}
		}
	}
	g.cells, g.spans = cells, spans
	return g
}

// slot is a placed cell: its node, the first row and column it covers and how
// many of each.
type slot struct {
	node       Node
	row, col   int
	cols, rows int
}

// slots places the cells in reading order, skipping covered places.
func (g gridNode) slots() []slot {
	n := len(g.tracks)
	var taken [][]bool // taken[row][col]
	free := func(r, c, cs, rs int) bool {
		for y := r; y < r+rs && y < len(taken); y++ {
			for x := c; x < c+cs; x++ {
				if taken[y][x] {
					return false
				}
			}
		}
		return true
	}
	mark := func(r, c, cs, rs int) {
		for len(taken) < r+rs {
			taken = append(taken, make([]bool, n))
		}
		for y := r; y < r+rs; y++ {
			for x := c; x < c+cs; x++ {
				taken[y][x] = true
			}
		}
	}
	out := make([]slot, 0, len(g.cells))
	r, c := 0, 0
	for i, node := range g.cells {
		sp := cellSpan{1, 1}
		if i < len(g.spans) {
			sp = g.spans[i]
		}
		cs, rs := min(max(sp.cols, 1), n), max(sp.rows, 1)
		for {
			if c+cs > n {
				r, c = r+1, 0
				continue
			}
			if free(r, c, cs, rs) {
				break
			}
			c++
		}
		mark(r, c, cs, rs)
		out = append(out, slot{node: node, row: r, col: c, cols: cs, rows: rs})
		c += cs
	}
	return out
}

// spanRows is the number of rows the placed cells reach.
func spanRows(slots []slot) int {
	n := 0
	for _, s := range slots {
		n = max(n, s.row+s.rows)
	}
	return n
}

// spread adds deficit across idx, evenly, the first entries taking the
// remainder. It does nothing with no entries.
func spread(v []int, idx []int, deficit int) {
	if deficit <= 0 || len(idx) == 0 {
		return
	}
	each, extra := deficit/len(idx), deficit%len(idx)
	for k, i := range idx {
		v[i] += each
		if k < extra {
			v[i]++
		}
	}
}

// spanContentWidths is contentWidths for a grid with spans: single-column
// cells set their column's width, then each spanning cell widens the auto-sized
// columns it covers if they and the gaps between are too narrow for it.
func (g gridNode) spanContentWidths(slots []slot) []int {
	w := make([]int, len(g.tracks))
	open := Constraints{MaxW: Unbounded, MaxH: Unbounded}
	for _, s := range slots {
		if s.cols == 1 {
			w[s.col] = max(w[s.col], s.node.Measure(open).W)
		}
	}
	for _, s := range slots {
		if s.cols == 1 {
			continue
		}
		have := sum(w[s.col:s.col+s.cols]) + gapOf(g.colGap, s.cols)
		var auto []int
		for c := s.col; c < s.col+s.cols; c++ {
			if g.tracks[c].Size <= 0 {
				auto = append(auto, c)
			}
		}
		spread(w, auto, s.node.Measure(open).W-have)
	}
	return w
}

// spanRowHeights is rowHeights for a grid with spans, the same two passes.
func (g gridNode) spanRowHeights(slots []slot, widths []int) []int {
	h := make([]int, spanRows(slots))
	span := func(s slot) int { return sum(widths[s.col:s.col+s.cols]) + gapOf(g.colGap, s.cols) }
	for _, s := range slots {
		if s.rows == 1 {
			h[s.row] = max(h[s.row], s.node.Measure(Constraints{MaxW: span(s), MaxH: Unbounded}).H)
		}
	}
	for _, s := range slots {
		if s.rows == 1 {
			continue
		}
		have := sum(h[s.row:s.row+s.rows]) + gapOf(g.rowGap, s.rows)
		rows := make([]int, 0, s.rows)
		for r := s.row; r < s.row+s.rows; r++ {
			rows = append(rows, r)
		}
		spread(h, rows, s.node.Measure(Constraints{MaxW: span(s), MaxH: Unbounded}).H-have)
	}
	return h
}

// placedRects returns each slot's node with the rectangle it fills at the
// solved widths and heights.
func (g gridNode) placedRects(slots []slot, widths, heights []int) []child {
	colGap, rowGap := max(g.colGap, 0), max(g.rowGap, 0)
	xs, ys := make([]int, len(widths)), make([]int, len(heights))
	for i, x := 0, 0; i < len(widths); i++ {
		xs[i] = x
		x += widths[i] + colGap
	}
	for i, y := 0, 0; i < len(heights); i++ {
		ys[i] = y
		y += heights[i] + rowGap
	}
	out := make([]child, len(slots))
	for i, s := range slots {
		out[i] = child{s.node, Rect{
			X: xs[s.col], Y: ys[s.row],
			W: sum(widths[s.col:s.col+s.cols]) + gapOf(colGap, s.cols),
			H: sum(heights[s.row:s.row+s.rows]) + gapOf(rowGap, s.rows),
		}}
	}
	return out
}

// renderSpans composes the placed cells into text: each output row takes, from
// left to right, the line of every cell that covers it, with blank space where
// no cell does (gaps, and places the cells left empty).
func (g gridNode) renderSpans(slots []slot, widths, heights []int) string {
	rects := g.placedRects(slots, widths, heights)
	totalW := sum(widths) + gapOf(g.colGap, len(widths))
	totalH := sum(heights) + gapOf(g.rowGap, len(heights))
	type drawn struct {
		x, w  int
		lines []string
		y     int
	}
	cells := make([]drawn, 0, len(rects))
	for _, r := range rects {
		if r.rect.W <= 0 || r.rect.H <= 0 {
			continue
		}
		cells = append(cells, drawn{x: r.rect.X, w: r.rect.W, y: r.rect.Y, lines: strings.Split(r.node.Render(Size{W: r.rect.W, H: r.rect.H}), "\n")})
	}
	rows := make([]string, totalH)
	for y := range rows {
		var on []drawn
		for _, c := range cells {
			if y >= c.y && y < c.y+len(c.lines) {
				on = append(on, c)
			}
		}
		sort.Slice(on, func(i, j int) bool { return on[i].x < on[j].x })
		var b strings.Builder
		x := 0
		for _, c := range on {
			if c.x < x {
				continue
			}
			b.WriteString(strings.Repeat(" ", c.x-x))
			line := c.lines[y-c.y]
			b.WriteString(line)
			b.WriteString(strings.Repeat(" ", max(c.w-ansi.Width(line), 0)))
			x = c.x + c.w
		}
		b.WriteString(strings.Repeat(" ", max(totalW-x, 0)))
		rows[y] = b.String()
	}
	return strings.Join(rows, "\n")
}
