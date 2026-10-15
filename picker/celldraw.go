package picker

import (
	"strconv"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/fuzzy"
	"github.com/ows4444/tui/internal/celldraw"
)

// DrawCells draws the list into the region r of buf, showing what View shows
// (it implements tui.CellDrawer): the query line, the rows with the cursor
// marker and the matched runes underlined, written straight into the grid with
// no row string built. A label that holds control characters is drawn through
// View instead.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	sub := buf.Sub(r)
	if sub.Width() == 0 || sub.Height() == 0 {
		return
	}
	sel := m.themed().ResolvedStates().Selected.Bold()
	faint, ok1 := celldraw.StyleID(sub, ansi.NewStyle().Faint())
	selID, ok2 := celldraw.StyleID(sub, sel)
	ulID, ok3 := celldraw.StyleID(sub, ansi.NewStyle().Underline())
	selUL, ok4 := celldraw.StyleID(sub, sel.Underline())
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
		cursor := row.Index == m.cursor
		prefix := "  "
		base, hl := plain, ulID
		if cursor {
			prefix, base, hl = "> ", selID, selUL
		}
		x := sub.SetString(0, y, prefix, plain)
		for _, run := range fuzzy.Split(m.Items[row.Index].Label, row.Positions) {
			id := base
			if run.Match {
				id = hl
			}
			x += sub.SetString(x, y, run.Text, id)
		}
		y++
	}
}
