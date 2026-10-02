package contextmenu

import "github.com/ows4444/tui/layout"

// LayoutNode returns the menu's bordered box as an ordinary layout.Node (it
// is also a layout.CellNode, so it draws straight into a cell grid), cut to
// the size it is given. While the menu is closed it measures as nothing and
// renders blank. The Model is not changed.
func (m Model) LayoutNode() layout.Node {
	if !m.open {
		return layout.Block("")
	}
	return m.box()
}
