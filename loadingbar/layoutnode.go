package loadingbar

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the bar to a layout.Node. Measure reports its own Width
// by one row; Render draws the bar across the allotted width, using it in
// place of Width (rows beyond the first are blank). The Model is not changed.
func (m Model) LayoutNode() layout.Node { return barNode{m} }

type barNode struct{ m Model }

func (n barNode) Measure(c layout.Constraints) layout.Size {
	if n.m.Width <= 0 {
		return c.Constrain(layout.Size{})
	}
	return c.Constrain(layout.Size{W: n.m.Width, H: 1})
}

func (n barNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	m.Width = s.W
	return layout.Block(m.View()).Render(s)
}
