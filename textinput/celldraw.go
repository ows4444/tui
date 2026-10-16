package textinput

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/internal/celldraw"
)

// DrawCells draws the field into the first row of the region r of buf, showing
// the same cells View shows (prompt, value or placeholder, cursor and selection,
// the Width window included) without building View's string. It makes Model a
// tui.CellDrawer: a Program whose root is, or composes through tui.DrawChild, a
// Model draws it straight into the frame's cell grid. Drawing is clipped to r. A
// prompt, placeholder or value that holds control characters, or a style the
// grid cannot hold, is drawn through View instead.
//
// A type that embeds Model and changes what View shows (passwordinput masks,
// inputgroup adds a prefix) must define its own DrawCells too, or the promoted
// one would draw the unmasked field; each of them does.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	sub := buf.Sub(r)
	if sub.Width() == 0 || sub.Height() == 0 {
		return
	}
	text, ok1 := celldraw.StyleID(sub, m.TextStyle)
	cur, ok2 := celldraw.StyleID(sub, m.CursorStyle)
	hint, ok3 := celldraw.StyleID(sub, m.PlaceholderStyle)
	selID, ok4 := celldraw.StyleID(sub, m.selStyle())
	if !ok1 || !ok2 || !ok3 || !ok4 || !celldraw.Plain(m.Prompt, m.Placeholder, m.Value()) {
		tui.DrawView(buf, r, m.View())
		return
	}
	show := m.focused && m.cursorVisible
	x := sub.SetString(0, 0, m.Prompt, 0)

	n := m.ed.Len()
	if n == 0 {
		if show {
			x += sub.SetString(x, 0, " ", cur)
		}
		sub.SetString(x, 0, m.Placeholder, hint)
		return
	}
	start, end := m.window()
	at := m.ed.Cursor()
	lo, hi, selected := m.ed.Selection()
	for i := start; i < end; i++ {
		id := text
		switch {
		case show && i == at:
			id = cur
		case selected && i >= lo && i < hi:
			id = selID
		}
		x += sub.SetString(x, 0, m.ed.Cluster(i), id)
	}
	if show && at == n && at == end {
		sub.SetString(x, 0, " ", cur)
	}
}
