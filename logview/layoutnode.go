package logview

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the log to a layout.Node by way of its viewport, sized
// to the allotted Size. A log that was following its tail (scrolled to the
// bottom) keeps following it at the new size, as Append does; one the user
// scrolled up keeps its offset. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return logNode{m} }

type logNode struct{ m Model }

func (n logNode) Measure(c layout.Constraints) layout.Size {
	return n.m.Viewport.LayoutNode().Measure(c)
}

func (n logNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	vp := n.m.Viewport
	followTail := vp.AtBottom()
	vp.Width, vp.Height = s.W, s.H
	if followTail {
		vp.GotoBottom()
	}
	return vp.LayoutNode().Render(s)
}
