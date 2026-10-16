package menu

import "github.com/ows4444/tui/cellbuf"

// DrawCells draws the current level into the region r of buf, showing what
// View shows (it implements tui.CellDrawer): the level is a picker, which
// draws its rows straight into the grid.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	m.level().DrawCells(buf, r)
}
