package layout

// Rect is a rectangle of terminal cells: its top-left corner (X, Y) and its
// size, relative to the top-left corner of the root that Rects was given.
type Rect struct{ X, Y, W, H int }

// Empty reports whether r covers no cells.
func (r Rect) Empty() bool { return r.W <= 0 || r.H <= 0 }

// Contains reports whether the cell (x, y) is inside r.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// Local converts screen cell (x, y) to coordinates relative to r's top-left
// corner. It does not check that the cell is inside r.
func (r Rect) Local(x, y int) (lx, ly int) { return x - r.X, y - r.Y }

// CutTop splits r into its top h rows and the rest. h is clamped to
// 0..r.H, so the two parts always tile r exactly.
func (r Rect) CutTop(h int) (top, rest Rect) {
	h = clampInt(h, 0, max(r.H, 0))
	return Rect{r.X, r.Y, r.W, h}, Rect{r.X, r.Y + h, r.W, max(r.H, 0) - h}
}

// CutBottom splits r into its bottom h rows and the rest above them.
func (r Rect) CutBottom(h int) (bottom, rest Rect) {
	h = clampInt(h, 0, max(r.H, 0))
	return Rect{r.X, r.Y + max(r.H, 0) - h, r.W, h}, Rect{r.X, r.Y, r.W, max(r.H, 0) - h}
}

// CutLeft splits r into its left w columns and the rest.
func (r Rect) CutLeft(w int) (left, rest Rect) {
	w = clampInt(w, 0, max(r.W, 0))
	return Rect{r.X, r.Y, w, r.H}, Rect{r.X + w, r.Y, max(r.W, 0) - w, r.H}
}

// CutRight splits r into its right w columns and the rest to their left.
func (r Rect) CutRight(w int) (right, rest Rect) {
	w = clampInt(w, 0, max(r.W, 0))
	return Rect{r.X + max(r.W, 0) - w, r.Y, w, r.H}, Rect{r.X, r.Y, max(r.W, 0) - w, r.H}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// intersect returns the overlap of r and o; an empty Rect at r's corner when
// they do not overlap.
func (r Rect) intersect(o Rect) Rect {
	x0, y0 := max(r.X, o.X), max(r.Y, o.Y)
	x1, y1 := min(r.X+r.W, o.X+o.W), min(r.Y+r.H, o.Y+o.H)
	if x1 <= x0 || y1 <= y0 {
		return Rect{X: r.X, Y: r.Y}
	}
	return Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// Placed is one node of a laid-out tree with the cells it occupies.
type Placed struct {
	// Node is the node, as it appears in the tree.
	Node Node
	// Name is the label given by Named, or "".
	Name string
	// Rect is where the node is drawn, relative to the root. It is clipped
	// to every ancestor, so a child that overflows its parent reports only
	// the visible part (an empty Rect if none of it shows).
	Rect Rect
	// Depth is 0 for the root, 1 for its children, and so on.
	Depth int
}

// Rects returns every node of the tree under root, laid out at size s, in
// depth-first pre-order (a parent before its children, siblings in order),
// with the cells each occupies. s is the size root is rendered at: pass
// root.Measure(c), the size Draw uses, or the Size given to Render.
//
// The rectangles come from the same sizing that Render uses, so a position
// here is where that node's content appears in Render(s). Use it to build
// mouse regions (hittest.Map) from the layout instead of working the numbers out
// by hand. Coordinates are relative to root's top-left; add the view's own
// origin for terminal coordinates (in inline mode those are terminal-absolute).
//
// Row, Column, BoxNode, GridNode and Named containers report their children;
// every other Node, including widget adapters, is a leaf. A node that
// appears twice in the tree is reported twice.
func Rects(root Node, s Size) []Placed {
	var out []Placed
	full := Rect{W: s.W, H: s.H}
	walk(root, full, full, "", 0, &out)
	return out
}

// RectOf returns the Rect of the first node named name (see Named) in the
// tree under root laid out at size s, and false if there is none.
func RectOf(root Node, s Size, name string) (Rect, bool) {
	if name == "" {
		return Rect{}, false
	}
	for _, p := range Rects(root, s) {
		if p.Name == name {
			return p.Rect, true
		}
	}
	return Rect{}, false
}

// Named returns n with a label that Rects reports in Placed.Name, so an app
// can find the region a particular widget occupies with RectOf. It changes
// nothing about layout: Measure and Render pass straight through. An empty
// name is not searchable.
func Named(name string, n Node) Node { return named{name: name, node: n} }

type named struct {
	name string
	node Node
}

func (n named) Measure(c Constraints) Size { return n.node.Measure(c) }
func (n named) Render(s Size) string       { return n.node.Render(s) }

// child is a node and its unclipped rectangle relative to its parent.
type child struct {
	node Node
	rect Rect
}

// container is implemented by the built-in nodes that lay out children.
type container interface {
	children(s Size) []child
}

func (n named) children(s Size) []child {
	return []child{{node: n.node, rect: Rect{W: s.W, H: s.H}}}
}

// walk appends n and its descendants. abs is n's unclipped rectangle in root
// coordinates (children are placed from it); clip is the visible area
// inherited from the ancestors.
func walk(n Node, abs, clip Rect, name string, depth int, out *[]Placed) {
	if nm, ok := n.(named); ok {
		name = nm.name
	}
	*out = append(*out, Placed{Node: n, Name: name, Rect: abs.intersect(clip), Depth: depth})
	c, ok := n.(container)
	if !ok || abs.Empty() {
		return
	}
	for _, k := range c.children(Size{W: abs.W, H: abs.H}) {
		kabs := Rect{X: abs.X + k.rect.X, Y: abs.Y + k.rect.Y, W: k.rect.W, H: k.rect.H}
		// A named wrapper reports its own name on its own entry only.
		walk(k.node, kabs, abs.intersect(clip), childName(k.node), depth+1, out)
	}
}

// childName is the label a node carries itself, "" for unnamed nodes, so a
// name does not leak from a parent to its children.
func childName(n Node) string {
	if nm, ok := n.(named); ok {
		return nm.name
	}
	return ""
}

func (f flexNode) children(s Size) []child {
	if len(f.kids) == 0 {
		return nil
	}
	pl := f.plan(s)
	out := make([]child, len(f.kids))
	pos := pl.lead
	for i, k := range f.kids {
		off, cs := f.crossPlacement(k, pl.sizes[i], pl.crossSize)
		if f.row {
			out[i] = child{k.Node, Rect{X: pos, Y: off, W: pl.sizes[i], H: cs}}
		} else {
			out[i] = child{k.Node, Rect{X: off, Y: pos, W: cs, H: pl.sizes[i]}}
		}
		pos += pl.sizes[i] + pl.gap
	}
	return out
}

func (b boxNode) children(s Size) []child {
	ch := b.chrome()
	inner := Size{W: s.W - ch.W, H: s.H - ch.H}
	if inner.W <= 0 || inner.H <= 0 {
		return []child{{node: b.child, rect: Rect{}}}
	}
	x, y := b.box.padLeft, b.box.padTop
	if hasBorder(b.box.border) {
		x++
		y++
	}
	return []child{{node: b.child, rect: Rect{X: x, Y: y, W: inner.W, H: inner.H}}}
}

func (g gridNode) children(s Size) []child {
	if len(g.tracks) == 0 || len(g.cells) == 0 {
		return nil
	}
	widths, heights := g.sizes(s)
	colGap, rowGap := max(g.colGap, 0), max(g.rowGap, 0)
	var out []child
	y := 0
	for r, h := range heights {
		x := 0
		for c := range g.tracks {
			if i := r*len(g.tracks) + c; i < len(g.cells) {
				out = append(out, child{g.cells[i], Rect{X: x, Y: y, W: widths[c], H: h}})
			}
			x += widths[c] + colGap
		}
		y += h + rowGap
	}
	return out
}

// NamedAt returns the name of the innermost Named node whose Rect contains the
// cell (x, y) in the tree under root laid out at size s, and (x, y) relative
// to that node's top-left corner. ok is false when the cell is outside every
// Named node, so an app can route a click with hit-testing free of a second
// region list. Coordinates are as in Rects. When Named nodes overlap without
// nesting, the later one in Rects order wins.
func NamedAt(root Node, s Size, x, y int) (name string, relX, relY int, ok bool) {
	var best Placed
	for _, p := range Rects(root, s) {
		if p.Name != "" && p.Rect.Contains(x, y) {
			best, ok = p, true
		}
	}
	if !ok {
		return "", 0, 0, false
	}
	return best.Name, x - best.Rect.X, y - best.Rect.Y, true
}
