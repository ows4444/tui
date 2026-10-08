package radiogroup

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the group to a layout.Node: one row per option, or one
// row for a Horizontal group. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return layout.Block(m.View()) }
