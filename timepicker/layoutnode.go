package timepicker

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the time picker to a layout.Node on one row.
// The Model is not changed.
func (m Model) LayoutNode() layout.Node { return layout.Block(m.View()) }
