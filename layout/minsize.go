package layout

// MinSize returns a Node that renders n only when it has at least min to work
// with, and fallback otherwise — a "terminal too small" screen instead of a
// clipped frame. A min axis of zero (or less) is never too small. A nil
// fallback renders blank space of the allotted size.
//
// Render(s) draws fallback when s.W < min.W or s.H < min.H. Measure follows
// the same rule against the parent's maximum: when c.MaxW or c.MaxH cannot
// hold min, it measures fallback, so the two passes agree. Rects reports
// whichever of the two is showing.
func MinSize(n Node, min Size, fallback Node) Node {
	if fallback == nil {
		fallback = Block("")
	}
	return minSizeNode{node: n, min: min, fallback: fallback}
}

type minSizeNode struct {
	node, fallback Node
	min            Size
}

func (m minSizeNode) tooSmall(w, h int) bool {
	return w < m.min.W || h < m.min.H
}

func (m minSizeNode) pick(w, h int) Node {
	if m.tooSmall(w, h) {
		return m.fallback
	}
	return m.node
}

func (m minSizeNode) Measure(c Constraints) Size {
	return m.pick(c.MaxW, c.MaxH).Measure(c)
}

func (m minSizeNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	return m.pick(s.W, s.H).Render(s)
}

func (m minSizeNode) children(s Size) []child {
	return []child{{node: m.pick(s.W, s.H), rect: Rect{W: s.W, H: s.H}}}
}
