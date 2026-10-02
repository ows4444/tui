package skeleton

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the skeleton to a layout.Node. Measure reports its own
// Width by Lines; Render fills exactly the allotted Size, using it in place of
// Width and Lines, at the current shimmer frame. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return skeletonNode{m} }

type skeletonNode struct{ m Model }

func (n skeletonNode) Measure(c layout.Constraints) layout.Size {
	return c.Constrain(layout.Size{W: max(n.m.Width, 0), H: max(n.m.Lines, 0)})
}

func (n skeletonNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	m.Width, m.Lines = s.W, s.H
	return layout.Block(m.View()).Render(s)
}
