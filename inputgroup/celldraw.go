package inputgroup

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/cellbuf"
)

// DrawCells draws the field through View, so it shows what View shows. It is
// defined here, and not left to the embedded textinput.Model, because the
// promoted method would draw the embedded field and not this type's View.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	tui.DrawView(buf, r, m.View())
}
