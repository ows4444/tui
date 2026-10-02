package viewport

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the viewport to a layout.Node. Measure reports the
// content's natural size (widest line by number of lines). Render shows
// exactly the allotted Size: it uses that size in place of Width and Height,
// keeps the current scroll offsets (re-clamped to the new size), and does
// not change the Model, so the size a parent gives it never overwrites the
// Model's own Width and Height.
func (m Model) LayoutNode() layout.Node { return viewportNode{m} }

type viewportNode struct{ m Model }

func (n viewportNode) Measure(c layout.Constraints) layout.Size {
	return c.Constrain(layout.Size{W: n.m.maxLineWidth, H: len(n.m.lines)})
}

func (n viewportNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m // a copy: sizing it here must not leak back
	m.Width, m.Height = s.W, s.H
	m.setYOffset(m.yOffset)
	m.setXOffset(m.xOffset)
	return layout.Block(m.View()).Render(s)
}
