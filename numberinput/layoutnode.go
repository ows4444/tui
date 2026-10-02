package numberinput

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the field to a layout.Node. It is the embedded
// textinput.Model's node (natural size on one row, exact size on Render); the
// wrapper filters typed runes only, so View and layout are unchanged.
func (m Model) LayoutNode() layout.Node { return m.Model.LayoutNode() }
