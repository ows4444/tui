package avatar

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the avatar to a layout.Node. Measure reports Width by
// Height; Render draws at exactly the allotted Size, using it in place of
// Width and Height. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return avatarNode{m} }

type avatarNode struct{ m Model }

func (n avatarNode) Measure(c layout.Constraints) layout.Size {
	return c.Constrain(layout.Size{W: max(n.m.Width, 0), H: max(n.m.Height, 0)})
}

func (n avatarNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	m.Width, m.Height = s.W, s.H
	return m.View()
}
