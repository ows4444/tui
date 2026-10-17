package datatable

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/widgets"
)

// LayoutNode adapts the table to a layout.Node. Measure reports its natural
// size: the columns at their content widths (two spaces apart) by one row per
// record plus the header and divider. Render fits the table to the allotted
// Size: when too narrow it shrinks the widest columns first and truncates
// cells with an ellipsis, and when too short it pins the header and shows a
// window of rows that keeps the cursor row visible. The cursor row keeps its
// highlight; the Model is not changed.
//
// With FrozenCols set and the columns too wide for the Size, Render scrolls
// them as View does, and does not shrink them: the frozen columns stay at the
// left, and the others show from the column offset that keeps the current column
// in sight.
func (m Model) LayoutNode() layout.Node { return tableNode{m} }

type tableNode struct{ m Model }

// gap is the spacing widgets.TableRows puts between columns.
const gap = 2

// naturalWidths returns the width of each column: ColumnWidths[i] when that
// is positive, otherwise the widest header or cell, served from the cache.
func (n tableNode) naturalWidths() []int {
	m := n.m
	fixed := func(i int) int {
		if i < len(m.ColumnWidths) {
			return m.ColumnWidths[i]
		}
		return 0
	}
	w := make([]int, len(m.Headers))
	var nat []int
	for i := range w {
		if f := fixed(i); f > 0 {
			w[i] = f
			continue
		}
		if nat == nil {
			nat = n.measureShown()
		}
		w[i] = nat[i]
		col := m.column(i)
		if col.Max > 0 {
			w[i] = min(w[i], col.Max)
		}
		w[i] = max(w[i], col.Min)
	}
	return w
}

// formatScan bounds how many rows are measured to find the widths of formatted
// columns, which the width cache (keyed on the raw cells) cannot give.
const formatScan = 500

// measureShown is the natural width of each column: from the cache, or, when
// a Format changes what is drawn, from the text shown in the first rows.
func (n tableNode) measureShown() []int {
	m := n.m
	if !m.hasFormat() {
		return m.measure()
	}
	w := make([]int, len(m.Headers))
	for i, h := range m.Headers {
		w[i] = ansi.Width(ansi.Clean(m.Raw, h))
	}
	for _, row := range m.rowsIn(0, min(m.rowCount(), formatScan)) {
		for c, cell := range row {
			if c < len(w) {
				w[c] = max(w[c], ansi.Width(ansi.Clean(m.Raw, m.display(c, cell))))
			}
		}
	}
	return w
}

func sum(v []int) int {
	t := 0
	for _, x := range v {
		t += x
	}
	return t
}

func (n tableNode) Measure(c layout.Constraints) layout.Size {
	if len(n.m.Headers) == 0 {
		return c.Constrain(layout.Size{})
	}
	w := n.naturalWidths()
	return c.Constrain(layout.Size{W: cursorGutter + sum(w) + gap*(len(w)-1), H: n.m.rowCount() + 2 + n.m.top()})
}

// fitWidths shrinks the widest column one cell at a time until the columns
// plus their gaps fit total, never below one cell each.
func fitWidths(w []int, total int) []int {
	out := append([]int(nil), w...)
	over := sum(out) + gap*(len(out)-1) - total
	for ; over > 0; over-- {
		widest := 0
		for i := range out {
			if out[i] > out[widest] {
				widest = i
			}
		}
		if out[widest] <= 1 {
			break
		}
		out[widest]--
	}
	return out
}

// clip fits cell into width cells, ending in an ellipsis when it is cut.
func clip(cell string, width int, ellipsis string) string {
	if ansi.Width(cell) <= width {
		return cell
	}
	if width <= 1 {
		return ansi.Truncate(cell, width)
	}
	return ansi.Truncate(cell, width-1) + ellipsis
}

func (n tableNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	if len(m.Headers) == 0 {
		return layout.Block("").Render(s)
	}
	if m.top() > 0 {
		line := layout.Block(m.filterLine()).Render(layout.Size{W: s.W, H: 1})
		if s.H == 1 {
			return line
		}
		below := s
		below.H--
		sub := m
		sub.query, sub.editing = "", false // no second line from the node below
		return line + "\n" + tableNode{sub}.Render(below)
	}
	ellipsis := m.themed().GlyphSet().Ellipsis
	natural := n.naturalWidths()
	cols := n.visibleCols(natural, s.W)
	widths := natural
	if len(cols) == len(natural) {
		widths = fitWidths(natural, s.W-cursorGutter)
	} else {
		// Columns are scrolled, not shrunk: fit only the ones shown.
		picked := make([]int, len(cols))
		for i, c := range cols {
			picked[i] = natural[c]
		}
		picked = fitWidths(picked, s.W-cursorGutter)
		widths = append([]int(nil), natural...)
		for i, c := range cols {
			widths[c] = picked[i]
		}
	}

	headers := make([]string, len(m.Headers))
	for i, h := range m.Headers {
		headers[i] = clip(ansi.Clean(m.Raw, h), widths[i], ellipsis)
	}

	// Window of body rows: the header and divider take two rows.
	body := s.H - 2
	if body < 0 {
		body = 0
	}
	start, end := 0, m.rowCount()
	if m.rowCount() > body {
		start = m.cursor - body/2
		if max := m.rowCount() - body; start > max {
			start = max
		}
		if start < 0 {
			start = 0
		}
		end = start + body
	}
	rows := make([][]string, 0, end-start)
	for _, row := range m.rowsIn(start, end) {
		cells := make([]string, len(row))
		for i, cell := range row {
			if i < len(widths) {
				cells[i] = clip(ansi.Clean(m.Raw, m.display(i, cell)), widths[i], ellipsis)
			} else {
				cells[i] = ansi.Clean(m.Raw, cell)
			}
		}
		rows = append(rows, cells)
	}

	noop := func() {}
	rows = m.styleCells(rows, start, &noop)
	headers, rows = m.alignRows(headers, rows, widths)
	if len(cols) != len(m.Headers) {
		headers, rows = pickColumns(headers, rows, cols)
	}

	headerStyle := ansi.NewStyle().Bold().Foreground(m.themed().Primary)
	dividerStyle := ansi.NewStyle().Foreground(m.themed().Muted)
	cursorStyle := m.themed().ResolvedStates().Selected.Bold()
	out := widgets.TableRowsWith(headers, rows, headerStyle, dividerStyle, func(i int) ansi.Style {
		if !m.CellMode && start+i == m.cursor {
			return cursorStyle
		}
		return ansi.Style{}
	}, m.themed())
	return layout.Block(strings.TrimRight(withGutter(out, start, m.cursor), "\n")).Render(s)
}

// visibleCols returns the columns Render draws at width w: all of them, unless
// FrozenCols is set and they do not fit, when it is the frozen columns followed by
// the scrolling ones from the offset that keeps the current column in sight (as
// Update keeps it for View).
func (n tableNode) visibleCols(natural []int, w int) []int {
	m := n.m
	all := make([]int, len(natural))
	for i := range all {
		all[i] = i
	}
	if m.frozen() == 0 || cursorGutter+sum(natural)+gap*(len(natural)-1) <= w {
		return all
	}
	sub := m
	sub.Width = w
	f := sub.frozen()
	off := max(sub.colOffset, f)
	if sub.col >= f && sub.col < off {
		off = sub.col
	}
	for off < sub.col && sub.col >= off+sub.fit(off) {
		off++
	}
	sub.colOffset = off
	return sub.colWindow()
}

// pickColumns keeps only the columns cols of the header and every row.
func pickColumns(headers []string, rows [][]string, cols []int) ([]string, [][]string) {
	h := make([]string, len(cols))
	for i, c := range cols {
		h[i] = headers[c]
	}
	out := make([][]string, len(rows))
	for r, row := range rows {
		cells := make([]string, len(cols))
		for i, c := range cols {
			if c < len(row) {
				cells[i] = row[c]
			}
		}
		out[r] = cells
	}
	return h, out
}
