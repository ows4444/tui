package layout

// MinMax returns n with its size bounded rather than fixed. A zero (or
// negative) bound leaves that side open, so MinMax(n, Size{}, Size{}) is n
// itself. If a max is below its min on an axis, the min wins.
//
// Measure clamps n's own size into [min, max] on each axis, then applies the
// parent's constraints like every node. Render(s) draws n at s clamped into
// the same bounds, anchored at the top-left, then clips or pads the result
// to exactly s. Rects reports n at its real position, so Named nodes below a
// MinMax keep their rectangles.
func MinMax(n Node, min, max Size) Node {
	min.W, min.H = maxInt(min.W, 0), maxInt(min.H, 0)
	max.W, max.H = maxInt(max.W, 0), maxInt(max.H, 0)
	if min == (Size{}) && max == (Size{}) {
		return n
	}
	return minMaxNode{node: n, min: min, max: max}
}

func maxInt(a, b int) int {
	if b > a {
		return b
	}
	return a
}

type minMaxNode struct {
	node     Node
	min, max Size
}

func clampBound(v, lo, hi int) int {
	if hi > 0 && v > hi {
		v = hi
	}
	if lo > 0 && v < lo {
		v = lo
	}
	return v
}

func (m minMaxNode) clamp(s Size) Size {
	return Size{W: clampBound(s.W, m.min.W, m.max.W), H: clampBound(s.H, m.min.H, m.max.H)}
}

func (m minMaxNode) Measure(c Constraints) Size {
	return c.Constrain(m.clamp(m.node.Measure(c)))
}

func (m minMaxNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	return place(m.node.Render(m.clamp(s)), s.W, s.H, 0, 0)
}

func (m minMaxNode) children(s Size) []child {
	cs := m.clamp(s)
	return []child{{node: m.node, rect: Rect{W: cs.W, H: cs.H}}}
}
