package imageview

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the image to a layout.Node. Measure reports Width by
// Height; Render draws at exactly the allotted Size, using it in place of
// Width and Height. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return imageNode{m} }

type imageNode struct{ m Model }

func (n imageNode) Measure(c layout.Constraints) layout.Size {
	return c.Constrain(layout.Size{W: max(n.m.Width, 0), H: max(n.m.Height, 0)})
}

func (n imageNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	m.Width, m.Height = s.W, s.H
	return m.View()
}
