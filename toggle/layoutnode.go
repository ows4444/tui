package toggle

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the widget to a layout.Node of its own width on one row.
// The Model is not changed.
func (m Model) LayoutNode() layout.Node { return layout.Block(m.View()) }
