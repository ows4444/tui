package multiselect

import (
	"strconv"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/fuzzy"
	"github.com/ows4444/tui/internal/celldraw"
)

// DrawCells draws the list into the region r of buf, showing what View shows
// (it implements tui.CellDrawer): the query line, and each row with its cursor
// marker, checkbox and the matched runes underlined, written straight into the
// grid. A label that holds control characters is drawn through View instead.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	sub := buf.Sub(r)
	if sub.Width() == 0 || sub.Height() == 0 {
		return
	}
	faint, ok1 := celldraw.StyleID(sub, ansi.NewStyle().Faint())
	cur, ok2 := celldraw.StyleID(sub, m.themed().ResolvedStates().Selected.Bold())
	chk, ok3 := celldraw.StyleID(sub, ansi.NewStyle().Foreground(m.themed().Success))
	ul, ok4 := celldraw.StyleID(sub, ansi.NewStyle().Underline())
	plain := cellbuf.StyleID(0)
	labels := make([]string, len(m.Items))
	for i, it := range m.Items {
		labels[i] = it.Label
	}
	if !ok1 || !ok2 || !ok3 || !ok4 || !celldraw.Plain(append(labels, m.query)...) {
		tui.DrawView(buf, r, m.View())
		return
	}

	y := 0
	vis := m.visible()
	if m.Filter {
		x := sub.SetString(0, y, "/ "+m.query, plain)
		if m.query != "" {
			sub.SetString(x, y, " ("+strconv.Itoa(len(vis))+")", faint)
		}
		y++
		if len(vis) == 0 {
			sub.SetString(0, y, "  ", plain)
			sub.SetString(2, y, "no matches", faint)
			return
		}
	}
	for _, row := range vis {
		if y >= sub.Height() {
			return
		}
		if row.Index == m.cursor {
			sub.SetString(0, y, "> ", cur)
		} else {
			sub.SetString(0, y, "  ", plain)
		}
		if m.selected[row.Index] {
			sub.SetString(2, y, "[x]", chk)
		} else {
			sub.SetString(2, y, "[ ]", plain)
		}
		x := 6 // "> ", the box, and the space after it
		for _, run := range fuzzy.Split(m.Items[row.Index].Label, row.Positions) {
			id := plain
			if run.Match {
				id = ul
			}
			x += sub.SetString(x, y, run.Text, id)
		}
		y++
	}
}
