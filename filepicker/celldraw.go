package filepicker

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/internal/celldraw"
)

// DrawCells draws the visible entries into the region r of buf, showing what
// View shows (it implements tui.CellDrawer): the cursor marker, the entry name
// with "/" after a directory, and the entry under the cursor highlighted. A
// name that holds control characters, or a directory that could not be listed,
// is drawn through View instead.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	sub := buf.Sub(r)
	if sub.Width() == 0 || sub.Height() == 0 {
		return
	}
	cur, ok := celldraw.StyleID(sub, m.themed().ResolvedStates().Selected.Bold())
	names := make([]string, 0, len(m.entries))
	for _, e := range m.entries {
		names = append(names, e.Name)
	}
	if m.err != nil || !ok || !celldraw.Plain(names...) || m.Raw {
		tui.DrawView(buf, r, m.View())
		return
	}
	start, end := m.window()
	for i := start; i < end; i++ {
		y := i - start
		if y >= sub.Height() {
			return
		}
		e := m.entries[i]
		label := ansi.Clean(m.Raw, e.Name)
		if e.IsDir {
			label += "/"
		}
		if i == m.cursor {
			x := sub.SetString(0, y, "> ", 0)
			sub.SetString(x, y, label, cur)
		} else {
			sub.SetString(0, y, "  "+label, 0)
		}
	}
}
