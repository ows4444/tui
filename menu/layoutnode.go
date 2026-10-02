package menu

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the menu to a layout.Node by way of the picker that
// shows the level currently navigated into, so it fits the allotted Size and
// keeps the cursor item visible. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return m.level().LayoutNode() }
