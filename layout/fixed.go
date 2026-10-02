package layout

// Fixed returns n with an exact width, height or both. A zero (or negative)
// W or H leaves that axis to n, so Fixed(n, Size{W: 40}) fixes only the width
// and Fixed(n, Size{}) is n itself.
//
// Measure reports the fixed value on each fixed axis and n's own size on each
// free one, then applies the parent's constraints like every node. Render(s)
// draws n at the fixed size on each fixed axis and at s on each free one,
// anchored at the top-left, then clips or pads the result to exactly s: a
// Fixed larger than its space is clipped, a smaller one is padded with blanks.
// Rects reports n at its real position, so Named nodes below a Fixed keep
// their rectangles.
func Fixed(n Node, size Size) Node {
	size.W, size.H = max(size.W, 0), max(size.H, 0)
	if size == (Size{}) {
		return n
	}
	return fixedNode{node: n, size: size}
}

type fixedNode struct {
	node Node
	size Size
}

// resolve returns the size the child is drawn at inside a slot of s.
func (f fixedNode) resolve(s Size) Size {
	if f.size.W > 0 {
		s.W = f.size.W
	}
	if f.size.H > 0 {
		s.H = f.size.H
	}
	return s
}

func (f fixedNode) Measure(c Constraints) Size {
	own := f.node.Measure(c)
	if f.size.W > 0 {
		own.W = f.size.W
	}
	if f.size.H > 0 {
		own.H = f.size.H
	}
	return c.Constrain(own)
}

func (f fixedNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	return place(f.node.Render(f.resolve(s)), s.W, s.H, 0, 0)
}

func (f fixedNode) children(s Size) []child {
	cs := f.resolve(s)
	return []child{{node: f.node, rect: Rect{W: cs.W, H: cs.H}}}
}
