package layout

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// FlexChild is one child of a Row or Column, sized along the container's
// main axis (width for Row, height for Column). The zero value is a
// content-sized child that neither grows nor shrinks.
type FlexChild struct {
	Node Node
	// Basis is the preferred main-axis size in cells; 0 means "the size
	// the child measures to". A scrollable child (a viewport, a long
	// table) measures to its full content length, so to make one simply
	// fill the space left over, use Fill (Basis: 1 with Grow: 1); otherwise
	// a growing child that measures larger than the screen pushes its
	// siblings out of view.
	Basis int
	// BasisLen, when set (see Pct and Fr), takes precedence over Basis. Pct
	// sizes the child as a percentage of the main-axis space left after
	// gaps; Fr gives it no base size and a Grow share of n, so fr children
	// split whatever the fixed and percentage children leave. The zero
	// value is unset.
	BasisLen Length
	// Grow, when > 0, shares positive free space by weight.
	Grow int
	// Shrink, when > 0, absorbs overflow, weighted by the child's basis.
	Shrink int
	// Min is the smallest main-axis size the child may be given.
	Min int
	// Max is the largest main-axis size; 0 means unbounded.
	Max int
	// CrossAlign places the child on the container's cross axis (vertical
	// in a Row, horizontal in a Column). The zero value, CrossStretch,
	// gives the child the full cross size.
	CrossAlign CrossAlign
}

// CrossAlign is a per-child cross-axis placement for Row and Column.
type CrossAlign int

const (
	// CrossStretch renders the child at the container's full cross size.
	// It is the zero value and the behaviour Row and Column always had.
	CrossStretch CrossAlign = iota
	// CrossStart renders the child at its own measured cross size, flush
	// to the top of a Row or the left of a Column.
	CrossStart
	// CrossCenter is CrossStart centred on the cross axis; an odd
	// remainder goes after the child.
	CrossCenter
	// CrossEnd is CrossStart flush to the bottom of a Row or the right of
	// a Column.
	CrossEnd
)

// Row lays children out left to right, separated by gap columns. Children
// are stretched to the row's full height. Rendered at a Size it produces
// exactly that many columns and rows: space left over when no child grows
// is blank, and overflow that no child can shrink away is clipped by
// display width (never mid-rune or mid-escape).
func Row(gap int, children ...FlexChild) Node { return flexNode{row: true, gap: gap, kids: children} }

// RowJustify is Row that also distributes any space left after sizing along
// the main axis per justify (JustifyStart, the zero value, is plain Row).
// Space is only left when no child grows, or every growing child hit its
// Max.
func RowJustify(gap int, justify Justify, children ...FlexChild) Node {
	return flexNode{row: true, gap: gap, justify: justify, kids: children}
}

// ColumnJustify is Column with main-axis (vertical) Justify, as RowJustify
// is to Row.
func ColumnJustify(gap int, justify Justify, children ...FlexChild) Node {
	return flexNode{gap: gap, justify: justify, kids: children}
}

// Column lays children out top to bottom, separated by gap blank rows.
// Children are stretched to the column's full width. Sizing and clipping
// follow Row, with height as the main axis.
func Column(gap int, children ...FlexChild) Node {
	return flexNode{gap: gap, kids: children}
}

type flexNode struct {
	row     bool
	gap     int
	justify Justify
	kids    []FlexChild
	ar      *arena // scratch for one Draw; nil outside Draw
}

func (f flexNode) gapTotal() int {
	if f.gap <= 0 || len(f.kids) < 2 {
		return 0
	}
	return f.gap * (len(f.kids) - 1)
}

func (f flexNode) split(s Size) (main, cross int) {
	if f.row {
		return s.W, s.H
	}
	return s.H, s.W
}

func (f flexNode) join(main, cross int) Size {
	if f.row {
		return Size{W: main, H: cross}
	}
	return Size{W: cross, H: main}
}

// childConstraints bounds a child on the cross axis by cc and leaves the main
// axis open, so Measure reports its natural main size.
func (f flexNode) childConstraints(c Constraints) Constraints {
	if f.row {
		return Constraints{MaxW: Unbounded, MinH: 0, MaxH: c.MaxH}
	}
	return Constraints{MaxW: c.MaxW, MaxH: Unbounded}
}

func (f flexNode) Measure(c Constraints) Size {
	cc := f.childConstraints(c)
	main, cross := 0, 0
	for _, k := range f.kids {
		m := k.Node.Measure(cc)
		km, kc := f.split(m)
		if k.BasisLen.kind == lenFr {
			km = 0
		} else if k.BasisLen.kind == lenPct {
			km = pctOf(c.MaxW, c.MaxH, f.row, k.BasisLen.n)
		} else if k.Basis > 0 {
			km = k.Basis
		}
		main += clampAxis(km, k.Min, maxOrUnbounded(k.Max))
		if kc > cross {
			cross = kc
		}
	}
	return c.Constrain(f.join(main+f.gapTotal(), cross))
}

func maxOrUnbounded(m int) int {
	if m <= 0 {
		return Unbounded
	}
	return m
}

// flexPlan is the outcome of sizing a flex container at a final Size: each
// child's main-axis size, the cross-axis size every child is given, the
// separation between neighbours (the gap plus any justify space) and the
// leading offset before the first child. Render draws from it and Rects
// reads positions from it, so the two cannot disagree.
type flexPlan struct {
	sizes     []int
	crossSize int
	gap, lead int
}

func (f flexNode) plan(s Size) flexPlan {
	mainSize, crossSize := f.split(s)
	avail := mainSize - f.gapTotal()
	if avail < 0 {
		avail = 0
	}

	specs := specsFrom(f.ar, len(f.kids))
	cc := Constraints{MaxW: Unbounded, MaxH: Unbounded}
	if f.row {
		cc.MaxH = s.H
	} else {
		cc.MaxW = s.W
	}
	for i, k := range f.kids {
		basis, grow, shrink := k.Basis, k.Grow, k.Shrink
		switch k.BasisLen.kind {
		case lenPct:
			basis = int(int64(avail) * int64(k.BasisLen.n) / 100)
			if basis <= 0 {
				basis = 0
			}
		case lenFr:
			basis, grow, shrink = 0, k.BasisLen.n, 1
		default:
			if basis <= 0 {
				basis, _ = f.split(k.Node.Measure(cc))
			}
		}
		specs[i] = flexSpec{basis: basis, grow: grow, shrink: shrink, min: k.Min, max: maxOrUnbounded(k.Max)}
	}
	sizes := solveFlexIn(f.ar, avail, specs)

	gap, lead := f.gap, 0
	if f.gap < 0 {
		gap = 0
	}
	if f.justify != JustifyStart && len(f.kids) > 0 {
		if left := mainSize - sum(sizes) - f.gapTotal(); left > 0 {
			var between int
			lead, between, _ = justifyGaps(f.justify, len(f.kids), left)
			gap += between
		}
	}
	return flexPlan{sizes: sizes, crossSize: crossSize, gap: gap, lead: lead}
}

func (f flexNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	pl := f.plan(s)
	sizes, gap, lead := pl.sizes, pl.gap, pl.lead

	blocks := make([]string, len(f.kids))
	for i, k := range f.kids {
		blocks[i] = f.renderChild(k, sizes[i], pl.crossSize)
	}

	var joined string
	if f.row {
		joined = joinHorizontal(gap, blocks...)
		if lead > 0 {
			pad := strings.Repeat(" ", lead)
			joined = pad + strings.ReplaceAll(joined, "\n", "\n"+pad)
		}
	} else {
		joined = joinColumn(gap, blocks, sizes)
		if lead > 0 {
			joined = strings.Repeat("\n", lead) + joined
		}
	}
	// Pads leftover space and clips overflow so the result is exactly s.
	return Block(joined).Render(s)
}

// BoxNode wraps child in box's padding, border and margin as a Node. Measure
// adds the box's chrome to the child's size; Render gives the child what is
// left of the allotted Size after the chrome and draws the box around it.
func BoxNode(box Box, child Node) Node { return boxNode{box: box, child: child} }

type boxNode struct {
	box   Box
	child Node
}

func (b boxNode) chrome() Size {
	w := b.box.padLeft + b.box.padRight + b.box.marginL + b.box.marginR
	h := b.box.padTop + b.box.padBottom + b.box.marginT + b.box.marginB
	if hasBorder(b.box.border) {
		w += 2
		h += 2
	}
	return Size{W: w, H: h}
}

func shrinkBound(v, by int) int {
	if v >= Unbounded {
		return Unbounded
	}
	if v -= by; v < 0 {
		return 0
	}
	return v
}

func (b boxNode) Measure(c Constraints) Size {
	ch := b.chrome()
	inner := Constraints{
		MinW: shrinkBound(c.MinW, ch.W), MaxW: shrinkBound(c.MaxW, ch.W),
		MinH: shrinkBound(c.MinH, ch.H), MaxH: shrinkBound(c.MaxH, ch.H),
	}
	in := b.child.Measure(inner)
	return c.Constrain(Size{W: in.W + ch.W, H: in.H + ch.H})
}

func (b boxNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	ch := b.chrome()
	inner := Size{W: s.W - ch.W, H: s.H - ch.H}
	content := ""
	box := b.box
	if inner.W > 0 {
		box = box.minWidth(inner.W)
	}
	if inner.W > 0 && inner.H > 0 {
		content = b.child.Render(inner)
	} else {
		// No room for the child: draw the frame around exactly the rows
		// that are left, none included. Box.Render alone would give empty
		// content one blank row and push the bottom border off the end.
		box = box.exactRows(maxInt(inner.H, 0))
	}
	return Block(box.Render(content)).Render(s)
}

// joinColumn stacks blocks with gap blank rows between neighbours, like
// joinVertical, except that a block whose allotted height is 0 contributes no
// rows: joinVertical counts an empty block as one blank line, which would make
// Render taller than Measure and clip the last sibling. The gaps around a
// zero-height child are kept, as Measure counts them.
func joinColumn(gap int, blocks []string, heights []int) string {
	zero := false
	for _, h := range heights {
		if h <= 0 {
			zero = true
			break
		}
	}
	if !zero {
		return joinVertical(gap, blocks...) // the common case, unchanged
	}
	if gap < 0 {
		gap = 0
	}
	var rows []string
	for i, b := range blocks {
		if i > 0 {
			for g := 0; g < gap; g++ {
				rows = append(rows, "")
			}
		}
		if heights[i] > 0 {
			rows = append(rows, strings.Split(b, "\n")...)
		}
	}
	// A single block pads every row to the widest, as joinVertical does.
	return joinVertical(0, strings.Join(rows, "\n"))
}

// crossPlacement returns k's offset and size on the cross axis inside a slot
// crossSize cells across, per k.CrossAlign.
func (f flexNode) crossPlacement(k FlexChild, main, crossSize int) (off, cs int) {
	if k.CrossAlign == CrossStretch {
		return 0, crossSize
	}
	// Measure the child's own cross size within what it is allotted.
	limit := f.join(main, crossSize)
	m := k.Node.Measure(Constraints{MaxW: limit.W, MaxH: limit.H})
	_, cs = f.split(m)
	if cs > crossSize {
		cs = crossSize
	}
	if cs < 0 {
		cs = 0
	}
	switch k.CrossAlign {
	case CrossCenter:
		off = (crossSize - cs) / 2
	case CrossEnd:
		off = crossSize - cs
	}
	return off, cs
}

// renderChild renders k at main size main, placed on the cross axis of a
// crossSize-wide (Column) or crossSize-tall (Row) slot per k.CrossAlign.
func (f flexNode) renderChild(k FlexChild, main, crossSize int) string {
	off, cs := f.crossPlacement(k, main, crossSize)
	if k.CrossAlign == CrossStretch {
		return k.Node.Render(f.join(main, crossSize))
	}
	inner := k.Node.Render(f.join(main, cs))
	if f.row {
		return place(inner, main, crossSize, 0, off)
	}
	return place(inner, crossSize, main, off, 0)
}

// place returns block, which is w0 x h0 cells or smaller, on a blank w x h
// canvas with its top-left corner at (dx, dy).
func place(block string, w, h, dx, dy int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	lines := strings.Split(block, "\n")
	out := make([]string, h)
	blank := strings.Repeat(" ", w)
	for y := range out {
		out[y] = blank
	}
	for i, l := range lines {
		y := dy + i
		if y < 0 || y >= h {
			continue
		}
		l = ansi.Truncate(l, w-dx)
		right := w - dx - ansi.Width(l)
		if right < 0 {
			right = 0
		}
		out[y] = strings.Repeat(" ", dx) + l + strings.Repeat(" ", right)
	}
	return strings.Join(out, "\n")
}

type lenKind int

const (
	lenUnset lenKind = iota
	lenPct
	lenFr
)

// Length is a flexible main-axis size for FlexChild.BasisLen. The zero value
// is unset. Build one with Pct or Fr.
type Length struct {
	kind lenKind
	n    int
}

// Pct is a Length of n percent of the container's main-axis space after gaps.
func Pct(n int) Length { return Length{kind: lenPct, n: n} }

// Fr is a Length that shares the space left after fixed and percentage
// children in proportion to n (a value below 1 counts as 1).
func Fr(n int) Length {
	if n < 1 {
		n = 1
	}
	return Length{kind: lenFr, n: n}
}

// pctOf resolves a percentage against a Measure bound; an unbounded axis
// resolves to 0 since there is nothing to take a percentage of.
func pctOf(maxW, maxH int, row bool, pct int) int {
	m := maxH
	if row {
		m = maxW
	}
	if m >= Unbounded || m <= 0 || pct <= 0 {
		return 0
	}
	return int(int64(m) * int64(pct) / 100)
}

// Windowed is implemented by a Node that can render a slice of its rows
// without producing the rest — a very long list or log. Scroll uses it to ask
// the child for only the visible window.
type Windowed interface {
	Node
	// Rows is the total number of rows the node has at width w.
	Rows(w int) int
	// RenderRows returns rows [start, start+count) at width w, joined by
	// "\n". It is never asked for more rows than fit the viewport.
	RenderRows(w, start, count int) string
}

// Scroll returns a Node that shows a window of child starting at row offset.
// Measure reports the child's natural size (constrained); Render gives the
// child its full natural height and returns the s.H rows from offset, with
// offset clamped to [0, height-s.H] so the window never runs past the end.
// A child that implements Windowed is instead asked for only the s.H visible
// rows. Use it inside Fill for scrollable content. When child is a Column
// (not Windowed), its children wrapped in Sticky stay at the top of the
// viewport once scrolled past.
func Scroll(child Node, offset int) Node { return scrollNode{child: child, offset: offset} }

type scrollNode struct {
	child  Node
	offset int
}

func (n scrollNode) Measure(c Constraints) Size {
	if w, ok := n.child.(Windowed); ok {
		return c.Constrain(Size{W: w.Measure(Constraints{MaxW: c.MaxW, MaxH: Unbounded}).W, H: w.Rows(c.MaxW)})
	}
	return c.Constrain(n.child.Measure(Constraints{MaxW: c.MaxW, MaxH: Unbounded}))
}

func (n scrollNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	if w, ok := n.child.(Windowed); ok {
		off := n.offset
		if max := w.Rows(s.W) - s.H; off > max {
			off = max
		}
		if off < 0 {
			off = 0
		}
		return Block(w.RenderRows(s.W, off, s.H)).Render(s)
	}
	ps := n.pins(s.W, max(n.child.Measure(Constraints{MaxW: s.W, MaxH: Unbounded}).H, s.H))
	full := n.child.Measure(Constraints{MaxW: s.W, MaxH: Unbounded})
	h := full.H
	if h < s.H {
		h = s.H
	}
	rows := strings.Split(n.child.Render(Size{W: s.W, H: h}), "\n")
	off := n.offset
	if max := len(rows) - s.H; off > max {
		off = max
	}
	if off < 0 {
		off = 0
	}
	view := Block(strings.Join(rows[off:], "\n")).Render(s)
	if ps != nil {
		view = overlayPins(view, ps, off, s.W, s.H)
	}
	return view
}
