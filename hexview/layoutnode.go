package hexview

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the view to a layout.Node of its own size. The Model is not
// changed.
func (m Model) LayoutNode() layout.Node { return layout.Block(m.View()) }
