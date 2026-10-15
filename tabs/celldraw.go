package tabs

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/internal/celldraw"
)

// DrawCells draws the bar into the first row of region r of buf, showing what
// View shows (it implements tui.CellDrawer). A label that holds control
// characters is drawn through View instead.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	sub := buf.Sub(r)
	if sub.Width() == 0 || sub.Height() == 0 {
		return
	}
	active, ok1 := celldraw.StyleID(sub, m.themed().ResolvedStates().Selected.Bold())
	inactive, ok2 := celldraw.StyleID(sub, ansi.NewStyle().Foreground(m.themed().Muted))
	if !ok1 || !ok2 || !celldraw.Plain(m.Labels...) {
		tui.DrawView(buf, r, m.View())
		return
	}
	x := 0
	for i, l := range m.Labels {
		if i > 0 {
			x += sub.SetString(x, 0, " ", 0)
		}
		if i == m.active {
			x += sub.SetString(x, 0, "["+l+"]", active)
		} else {
			x += sub.SetString(x, 0, " "+l+" ", inactive)
		}
	}
}
