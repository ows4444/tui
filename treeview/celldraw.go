package treeview

import (
	"strconv"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/fuzzy"
	"github.com/ows4444/tui/internal/celldraw"
)

// DrawCells draws the visible rows into the region r of buf, showing what View
// shows (it implements tui.CellDrawer): the query line, and each row with its
// cursor marker, indent, expand marker and label with the matched runes
// underlined, written straight into the grid. A label that holds control
// characters is drawn through View instead.
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
	items := m.flatten()
	texts := make([]string, 0, len(items)+1)
	texts = append(texts, m.query)
	for _, it := range items {
		texts = append(texts, it.label)
	}
	if !ok1 || !ok2 || !ok3 || !ok4 || !celldraw.Plain(texts...) {
		tui.DrawView(buf, r, m.View())
		return
	}

	y := 0
	if m.Filter {
		x := sub.SetString(0, y, "/ "+m.query, plain)
		if m.query != "" {
			sub.SetString(x, y, " ("+strconv.Itoa(len(items))+")", faint)
		}
		y++
		if len(items) == 0 {
			if m.query != "" {
				sub.SetString(2, y, "no matches", faint)
			}
			return
		}
	}
	if len(items) == 0 {
		return
	}
	cursor := clamp(m.cursor, 0, len(items)-1)
	glyphs := m.themed().GlyphSet()
	for i, item := range items {
		if y >= sub.Height() {
			return
		}
		prefix, base, hl := "  ", plain, ulID
		if i == cursor {
			prefix, base, hl = "> ", selID, selUL
		}
		marker := " "
		if len(item.node.Children) > 0 {
			marker = glyphs.Collapsed
			if item.open {
				marker = glyphs.Expanded
			}
		}
		x := sub.SetString(0, y, prefix, plain)
		for d := 0; d < item.depth; d++ {
			x += sub.SetString(x, y, "  ", plain)
		}
		x += sub.SetString(x, y, marker+" ", plain)
		for _, run := range fuzzy.Split(item.label, item.pos) {
			id := base
			if run.Match {
				id = hl
			}
			x += sub.SetString(x, y, run.Text, id)
		}
		y++
	}
}
