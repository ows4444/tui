package button

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the button to a layout.Node of its own width on one row.
// The Model is not changed.
func (m Model) LayoutNode() layout.Node { return layout.Block(m.View()) }
