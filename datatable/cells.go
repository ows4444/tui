package datatable

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

// frozen is the number of leading columns that do not scroll: FrozenCols,
// kept below the number of columns so one always scrolls.
func (m Model) frozen() int {
	return max(min(m.FrozenCols, len(m.Headers)-1), 0)
}

// fit is how many scrolling columns, starting at first, fit in Width beside
// the frozen ones (at least one).
func (m Model) fit(first int) int {
	f := m.frozen()
	if f == 0 {
		return m.fitCols(first)
	}
	n := len(m.Headers)
	if m.Width <= 0 || first >= n {
		return max(n-first, 0)
	}
	start, cnt := m.window()
	width := func(c int) int {
		w := ansi.Width(m.Headers[c])
		for _, row := range m.rowsIn(start, start+cnt) {
			if c < len(row) {
				w = max(w, ansi.Width(row[c]))
			}
		}
		return w
	}
	total := cursorGutter
	for c := 0; c < f; c++ {
		if c > 0 {
			total += 2
		}
		total += width(c)
	}
	k := 0
	for c := first; c < n; c++ {
		w := width(c) + 2
		if k > 0 && total+w > m.Width {
			break
		}
		total += w
		k++
	}
	return max(k, 1)
}

// colWindow lists, in order, the columns that are drawn when Width limits the
// table: the frozen columns, then the scrolling ones from colOffset.
func (m Model) colWindow() []int {
	n := len(m.Headers)
	f := m.frozen()
	first := max(clamp(m.colOffset, 0, n-1), f)
	last := min(first+max(m.fit(first), 1), n)
	cols := make([]int, 0, f+last-first)
	for c := 0; c < f; c++ {
		cols = append(cols, c)
	}
	for c := first; c < last; c++ {
		cols = append(cols, c)
	}
	return cols
}

// styled reports whether a cell is drawn with a style of its own: CellStyle,
// or the cursor cell.
func (m Model) styled() bool { return m.CellStyle != nil || m.CellMode || m.FrozenCols > 0 }

// styleCells returns rows with CellStyle applied to each cell and, in CellMode,
// the cursor style on the cell under the cursor. own is called before a row is
// changed, so the caller's slices are copied, not written.
func (m Model) styleCells(rows [][]string, start int, own *func()) [][]string {
	if m.CellStyle == nil && !m.CellMode {
		return rows
	}
	cursorStyle := m.themed().ResolvedStates().Selected.Bold()
	(*own)()
	out := make([][]string, len(rows))
	for i, row := range rows {
		cells := make([]string, len(row))
		copy(cells, row)
		for c, v := range row {
			var st ansi.Style
			if m.CellStyle != nil {
				st = m.CellStyle(start+i, c, v)
			}
			if m.CellMode && start+i == m.cursor && c == m.col {
				st = cursorStyle
			}
			if st != (ansi.Style{}) {
				cells[c] = st.Render(v)
			}
		}
		out[i] = cells
	}
	return out
}

// drawViaView draws the View string, for the features DrawCells does not
// draw cell by cell.
func (m Model) drawViaView(buf *cellbuf.Buffer, r cellbuf.Rect) {
	tui.DrawView(buf, r, m.View())
}
