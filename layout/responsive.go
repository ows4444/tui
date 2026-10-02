package layout

// Break is one breakpoint of Responsive: its node is used while the
// available width is at most MaxW and the available height is at most MaxH.
// A MaxW or MaxH of zero or less admits any width or height respectively.
type Break struct{ MaxW, MaxH int }

// Responsive picks a node by the available size. breaks[i] pairs with
// nodes[i]: the first node whose Break admits both the width and the height is used. A node with
// no Break (nodes longer than breaks) is the fallback for larger sizes; if
// every break is exceeded and there is no such node, the last node is used.
// With no nodes it is empty.
//
// Measure uses the node chosen by the constraint's MaxW and MaxH, so the answer is
// the layout that Render will pick when given that width. Render(s) draws the
// node chosen for s at exactly s. Rects reports the chosen node at its real
// position, like Fixed.
func Responsive(breaks []Break, nodes ...Node) Node {
	return responsiveNode{breaks: append([]Break(nil), breaks...), nodes: append([]Node(nil), nodes...)}
}

type responsiveNode struct {
	breaks []Break
	nodes  []Node
}

// pick returns the node for an available size.
func (r responsiveNode) pick(s Size) Node {
	if len(r.nodes) == 0 {
		return block("")
	}
	for i, b := range r.breaks {
		if i >= len(r.nodes) {
			break
		}
		if (b.MaxW <= 0 || s.W <= b.MaxW) && (b.MaxH <= 0 || s.H <= b.MaxH) {
			return r.nodes[i]
		}
	}
	if len(r.nodes) > len(r.breaks) {
		return r.nodes[len(r.breaks)]
	}
	return r.nodes[len(r.nodes)-1]
}

func (r responsiveNode) Measure(c Constraints) Size {
	return c.Constrain(r.pick(Size{W: c.MaxW, H: c.MaxH}).Measure(c))
}

func (r responsiveNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	return place(r.pick(s).Render(s), s.W, s.H, 0, 0)
}

func (r responsiveNode) children(s Size) []child {
	return []child{{node: r.pick(s), rect: Rect{W: s.W, H: s.H}}}
}
