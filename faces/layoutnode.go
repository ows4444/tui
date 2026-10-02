package faces

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the face to a layout.Node. The Model's Size is the
// largest head it will use: Render draws the Large head only if it (and the
// label row, if ShowLabel) fits the allotted Size, and falls back to the
// Small head otherwise; a Small head that still does not fit is cut. Measure
// reports the size of the head the Model asks for. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return faceNode{m} }

type faceNode struct{ m Model }

func (n faceNode) cells(size Size) layout.Size {
	w, h := size.Cells()
	if n.m.ShowLabel {
		h++
	}
	return layout.Size{W: w, H: h}
}

func (n faceNode) Measure(c layout.Constraints) layout.Size {
	return c.Constrain(n.cells(n.m.Size))
}

func (n faceNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	if m.Size == Large {
		if need := n.cells(Large); need.W > s.W || need.H > s.H {
			m.Size = Small
		}
	}
	return layout.Block(m.View()).Render(s)
}
