package layout

import (
	"slices"
	"strings"
)

// Stack overlays its children on one another, all with their top-left corner
// at the Stack's own (0, 0) unless wrapped in Absolute. Children are drawn at
// their natural size (what Measure reports within the Stack's bounds), opaque
// over what is beneath them, in ascending z order: a child wrapped in
// Layer(z, n) with a higher z is drawn on top of one with a lower z where they
// overlap, and children of equal z (an unwrapped child has z 0) are drawn in
// the order given, so the later one is on top. Measure is the bounding box of
// the children. Layer and Absolute must be direct children of the Stack; a
// Named wrapper around them hides them from it. Stack is a CellNode, and
// Rects and NamedAt report its children in drawing order, so the topmost
// Named node is the last one that contains a cell.
func Stack(children ...Node) Node { return stackNode{kids: children} }

// Layer returns n tagged with stacking order z for a Stack: of two overlapping
// children, the one with the higher z is drawn on top. Outside a Stack it is
// n. Layer composes with Absolute in either order.
func Layer(z int, n Node) Node {
	l := asLayer(n)
	l.z = z
	return l
}

// Absolute returns n tagged to be placed with its top-left corner at column x,
// row y of the enclosing Stack (negative values count as 0) instead of at its
// origin. Outside a Stack it is n. Absolute composes with Layer in either
// order.
func Absolute(x, y int, n Node) Node {
	l := asLayer(n)
	l.x, l.y = max(x, 0), max(y, 0)
	return l
}

// layerNode carries a child's stacking order and offset for a Stack.
type layerNode struct {
	z, x, y int
	node    Node
}

// asLayer returns n as a layerNode, wrapping it when it is not one.
func asLayer(n Node) layerNode {
	if l, ok := n.(layerNode); ok {
		return l
	}
	return layerNode{node: n}
}

func (l layerNode) Measure(c Constraints) Size { return l.node.Measure(c) }
func (l layerNode) Render(s Size) string       { return l.node.Render(s) }
func (l layerNode) DrawCells(dst CellSurface, r Rect) {
	drawNode(l.node, dst, r)
}

func (l layerNode) children(s Size) []child {
	return []child{{node: l.node, rect: Rect{W: s.W, H: s.H}}}
}

type stackNode struct{ kids []Node }

// placedKid is one Stack child with where and how big it is drawn.
type placedKid struct {
	node Node
	rect Rect
	z    int
}

// layout returns the visible children within s in drawing order (bottom
// first), those with an empty rectangle omitted.
func (st stackNode) layout(s Size) []placedKid {
	out := make([]placedKid, 0, len(st.kids))
	for _, k := range st.kids {
		l := asLayer(k)
		sz := l.node.Measure(Constraints{MaxW: max(s.W-l.x, 0), MaxH: max(s.H-l.y, 0)})
		if sz.W <= 0 || sz.H <= 0 {
			continue
		}
		out = append(out, placedKid{node: l.node, rect: Rect{X: l.x, Y: l.y, W: sz.W, H: sz.H}, z: l.z})
	}
	slices.SortStableFunc(out, func(a, b placedKid) int { return a.z - b.z })
	return out
}

func (st stackNode) Measure(c Constraints) Size {
	var w, h int
	for _, k := range st.kids {
		l := asLayer(k)
		sz := l.node.Measure(Constraints{MaxW: shrinkBound(c.MaxW, l.x), MaxH: shrinkBound(c.MaxH, l.y)})
		w, h = max(w, l.x+sz.W), max(h, l.y+sz.H)
	}
	return c.Constrain(Size{W: w, H: h})
}

func (st stackNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	blank := strings.Repeat(" ", s.W)
	rows := make([]string, s.H)
	for i := range rows {
		rows[i] = blank
	}
	canvas := strings.Join(rows, "\n")
	for _, k := range st.layout(s) {
		over := Block(k.node.Render(Size{W: k.rect.W, H: k.rect.H})).Render(Size{W: k.rect.W, H: k.rect.H})
		canvas = Overlay(canvas, over, k.rect.X, k.rect.Y)
	}
	return Block(canvas).Render(s)
}

func (st stackNode) DrawCells(dst CellSurface, r Rect) {
	if r.Empty() {
		return
	}
	prev := pushClip(dst, r)
	for _, k := range st.layout(Size{W: r.W, H: r.H}) {
		kr := Rect{X: r.X + k.rect.X, Y: r.Y + k.rect.Y, W: k.rect.W, H: k.rect.H}
		clearRect(dst, kr)
		drawNode(k.node, dst, kr)
	}
	dst.SetClip(prev)
}

func (st stackNode) children(s Size) []child {
	ks := st.layout(s)
	out := make([]child, len(ks))
	for i, k := range ks {
		out[i] = child{node: k.node, rect: k.rect}
	}
	return out
}

// Sticky returns n marked as a sticky row for the Scroll it is a child of: n
// is a child of a Column that is the direct child of a Scroll (see Scroll),
// and while the Scroll is scrolled past it, n is drawn at the top of the
// viewport instead of scrolling away. Several sticky rows pin one under the
// other in order, and a sticky row hides what scrolls beneath it. Anywhere
// else Sticky is n.
func Sticky(n Node) Node { return stickyNode{node: n} }

type stickyNode struct{ node Node }

func (s stickyNode) Measure(c Constraints) Size { return s.node.Measure(c) }
func (s stickyNode) Render(sz Size) string      { return s.node.Render(sz) }
func (s stickyNode) DrawCells(dst CellSurface, r Rect) {
	drawNode(s.node, dst, r)
}
func (s stickyNode) children(sz Size) []child {
	return []child{{node: s.node, rect: Rect{W: sz.W, H: sz.H}}}
}

// pin is a sticky row of a Scroll's content: its node and its row and height
// within the content.
type pin struct {
	node Node
	y, h int
}

// flexOf returns the Column n is, looking through the frame's memo wrapper.
func flexOf(n Node) (flexNode, bool) {
	switch v := n.(type) {
	case *memoNode:
		return flexOf(v.n)
	case *flexNode:
		return *v, !v.row
	case flexNode:
		return v, !v.row
	}
	return flexNode{}, false
}

func isSticky(n Node) bool {
	switch v := n.(type) {
	case *memoNode:
		return isSticky(v.n)
	case stickyNode:
		return true
	}
	return false
}

// pins returns the sticky rows of n's child laid out w wide and h tall, or nil
// when the child is not a Column with sticky children (without allocating).
func (n scrollNode) pins(w, h int) []pin {
	f, ok := flexOf(n.child)
	if !ok {
		return nil
	}
	found := false
	for _, k := range f.kids {
		if isSticky(k.Node) {
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	pl := f.plan(Size{W: w, H: h})
	var out []pin
	pos := pl.lead
	for i, k := range f.kids {
		if isSticky(k.Node) {
			out = append(out, pin{node: k.Node, y: pos, h: pl.sizes[i]})
		}
		pos += pl.sizes[i] + pl.gap
	}
	return out
}

// pinnedRows returns, for each pin that has scrolled past the top of a viewport
// showing content from row off, its row in the viewport; the others get -1.
func pinnedRows(ps []pin, off int, buf []int) []int {
	buf = buf[:0]
	top := 0
	for _, p := range ps {
		if p.y-off < top {
			buf = append(buf, top)
			top += p.h
		} else {
			buf = append(buf, -1)
		}
	}
	return buf
}

// overlayPins draws the pinned rows over the viewport view (rendered, h rows).
func overlayPins(view string, ps []pin, off, w, viewH int) string {
	for i, y := range pinnedRows(ps, off, nil) {
		p := ps[i]
		if y < 0 || y >= viewH {
			continue
		}
		rows := strings.Split(Block(p.node.Render(Size{W: w, H: p.h})).Render(Size{W: w, H: p.h}), "\n")
		if len(rows) > viewH-y {
			rows = rows[:viewH-y]
		}
		view = Overlay(view, strings.Join(rows, "\n"), 0, y)
	}
	return view
}

var (
	_ CellNode = layerNode{}
	_ CellNode = stackNode{}
	_ CellNode = stickyNode{}
)
