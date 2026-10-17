package spinner

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/internal/celldraw"
)

// DrawCells draws the frame and label into the region r of buf, showing what
// View shows (it implements tui.CellDrawer). A frame or label that holds
// control characters is drawn through View instead.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	sub := buf.Sub(r)
	if sub.Width() == 0 || sub.Height() == 0 {
		return
	}
	frames := m.frames()
	if len(frames) == 0 {
		if !celldraw.Plain(m.Label) {
			tui.DrawView(buf, r, m.View())
			return
		}
		sub.SetString(0, 0, m.Label, 0)
		return
	}
	idx := m.frame
	if m.Clock != nil {
		idx = m.Clock.Frame()
	}
	frame := frames[idx%len(frames)]
	id, ok := celldraw.StyleID(sub, ansi.NewStyle().Foreground(m.themed().Primary))
	if !ok || !celldraw.Plain(frame, m.Label) {
		tui.DrawView(buf, r, m.View())
		return
	}
	x := sub.SetString(0, 0, frame, id)
	if m.Label != "" {
		sub.SetString(x, 0, " "+m.Label, 0)
	}
}
