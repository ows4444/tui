package datatable

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

// styleID returns the id in buf of the look st gives text, and false when the
// cell grid cannot represent it.
func styleID(buf *cellbuf.Buffer, st ansi.Style) (cellbuf.StyleID, bool) {
	p, err := cellbuf.Parse(st.Render("x"))
	if err != nil {
		return 0, false
	}
	return buf.StyleID(p.Style(p.At(0, 0).Style)), true
}

// DrawCells draws the table into the region r of buf, as View would show it (it
// implements tui.CellDrawer): the header, the divider and the visible rows with
// the cursor row highlighted, each cell written straight into the grid, with no
// row or table string built. Column widths come from the width cache when the
// window is the whole table and every cell is clean, and from the visible rows
// otherwise, as View computes them. With Raw set, or when a style cannot be
// drawn in the grid, it draws the View string instead.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	if len(m.Headers) == 0 {
		return
	}
	sub := buf.Sub(r)
	hid, ok1 := styleID(sub, ansi.NewStyle().Bold().Foreground(m.themed().Primary))
	did, ok2 := styleID(sub, ansi.NewStyle().Foreground(m.themed().Muted))
	cid, ok3 := styleID(sub, m.themed().ResolvedStates().Selected.Bold())
	if m.Raw || !ok1 || !ok2 || !ok3 {
		tui.DrawView(buf, r, m.View())
		return
	}

	start, n := m.window()
	rows := m.Rows[start : start+n]
	first, last := 0, len(m.Headers)
	if m.Width > 0 {
		first = clamp(m.colOffset, 0, len(m.Headers)-1)
		last = min(first+max(m.fitCols(first), 1), len(m.Headers))
	}

	var nat []int
	if n == len(m.Rows) {
		if w, clean := m.measureClean(); clean {
			nat = w
		}
	}
	arrow := " ^"
	if m.sortDesc {
		arrow = " v"
	}
	hdr := func(c int) string { return ansi.Clean(false, m.Headers[c]) }
	arrowAt := func(c int) bool { return m.sorted && m.sortCol == c }
	width := func(c int) int {
		hw := ansi.Width(hdr(c))
		if arrowAt(c) {
			hw += len(arrow)
		}
		if nat != nil {
			return max(nat[c], hw)
		}
		w := hw
		for _, row := range rows {
			if c < len(row) {
				w = max(w, ansi.Width(ansi.Clean(false, row[c])))
			}
		}
		return w
	}

	// Column widths, once: they set the divider and every row's cell positions.
	var stack [16]int
	widths := stack[:0]
	total := 0
	for c := first; c < last; c++ {
		w := width(c)
		widths = append(widths, w)
		if c > first {
			total += gap
		}
		total += w
	}

	// Header, then divider.
	x := cursorGutter
	for i, c := 0, first; c < last; i, c = i+1, c+1 {
		w := widths[i]
		sub.Fill(cellbuf.Rect{X: x, Y: 0, W: w, H: 1}, " ", hid)
		cols := sub.SetString(x, 0, hdr(c), hid)
		if arrowAt(c) {
			sub.SetString(x+cols, 0, arrow, hid)
		}
		x += w + gap
	}
	if rule := m.themed().GlyphSet().RuleH; rule != "" {
		if rw := ansi.Width(rule); rw > 0 {
			sub.Fill(cellbuf.Rect{X: cursorGutter, Y: 1, W: total * rw, H: 1}, rule, did)
		}
	}

	// Rows.
	for i, row := range rows {
		y := chrome + i
		if y >= sub.Height() {
			break
		}
		var id cellbuf.StyleID
		if start+i == m.cursor {
			id = cid
			sub.SetString(0, y, ">", 0)
		}
		x := cursorGutter
		for j, c := 0, first; c < last; j, c = j+1, c+1 {
			if id != 0 {
				sub.Fill(cellbuf.Rect{X: x, Y: y, W: widths[j], H: 1}, " ", id)
			}
			if c < len(row) {
				sub.SetString(x, y, ansi.Clean(false, row[c]), id)
			}
			x += widths[j] + gap
		}
	}
}

// Compile-time proof that Model draws into a cell grid; tui.CellDrawer is
// structural, so the interface is spelled out here.
var _ interface {
	DrawCells(*cellbuf.Buffer, cellbuf.Rect)
} = Model{}
