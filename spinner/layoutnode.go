package spinner

import (
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the spinner to a layout.Node: it measures and renders
// whatever View shows (for a clock, read afresh each time), cut to the
// allotted Size. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return layout.ViewFunc(m.View) }
