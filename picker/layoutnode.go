package picker

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the list to a layout.Node. Measure reports its natural
// size (widest "> label" by one row per item). Render fits the allotted Size:
// items are cut to the width, and when there are more items than rows it
// shows a window of items containing the cursor item. The Model is not
// changed.
func (m Model) LayoutNode() layout.Node { return listNode{m} }

type listNode struct{ m Model }

func (n listNode) Measure(c layout.Constraints) layout.Size {
	w := 0
	for _, it := range n.m.Items {
		if lw := 2 + ansi.Width(it.Label); lw > w {
			w = lw
		}
	}
	return c.Constrain(layout.Size{W: w, H: len(n.m.Items)})
}

func (n listNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	lines := strings.Split(n.m.View(), "\n")
	if len(n.m.Items) == 0 {
		lines = nil
	}
	if len(lines) > s.H {
		start := clamp(n.m.cursor-s.H/2, 0, len(lines)-s.H)
		lines = lines[start : start+s.H]
	}
	return layout.Block(strings.Join(lines, "\n")).Render(s)
}
