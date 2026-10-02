package tabs

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the tab bar to a layout.Node. Measure reports its natural
// size (every tab " label " with one space between, one row). Render fits the
// allotted Size: when the bar is wider than the space, it scrolls so the
// active tab is always visible (its right edge if it fits, its left edge if
// the tab itself is wider than the space). The Model is not changed.
func (m Model) LayoutNode() layout.Node { return tabsNode{m} }

type tabsNode struct{ m Model }

// span is the [start, end) columns tab i occupies on the bar.
func (n tabsNode) span(i int) (start, end int) {
	for j := 0; j < i; j++ {
		start += ansi.Width(n.m.Labels[j]) + 2 + 1 // " label " and the gap after it
	}
	return start, start + ansi.Width(n.m.Labels[i]) + 2
}

func (n tabsNode) Measure(c layout.Constraints) layout.Size {
	if len(n.m.Labels) == 0 {
		return c.Constrain(layout.Size{})
	}
	_, end := n.span(len(n.m.Labels) - 1)
	return c.Constrain(layout.Size{W: end, H: 1})
}

func (n tabsNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	bar := n.m.View()
	if len(n.m.Labels) > 0 && ansi.Width(bar) > s.W {
		active := clamp(n.m.active, 0, len(n.m.Labels)-1)
		start, end := n.span(active)
		offset := end - s.W // put the active tab's right edge at the right side
		if offset > start {
			offset = start // ...unless the tab is wider than the space
		}
		if offset > 0 {
			bar = ansi.TrimLeftWidth(bar, offset)
		}
	}
	return layout.Block(bar).Render(s)
}
