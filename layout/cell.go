package layout

import "github.com/ows4444/tui/ansi"

// CellSurface is the grid a CellNode draws into. It is deliberately small and
// uses only layout types so that this package does not depend on any cell
// buffer: package cellbuf adapts a *cellbuf.Buffer with cellbuf.Layout.
// Coordinates are the surface's own, with (0, 0) at its top-left cell.
type CellSurface interface {
	// Clip returns the rectangle drawing is currently clipped to.
	Clip() Rect
	// SetClip clips later drawing to r, itself clipped to the surface.
	SetClip(r Rect)
	// Put writes s, a single-row string that may carry SGR styling, from
	// column x of row y, clipped to the clip rectangle, and returns the
	// columns written. It never wraps and writes nothing for "".
	Put(x, y int, s string) int
	// Repeat writes n copies of the one-cluster (possibly styled) string
	// cluster side by side from column x of row y, clipped to the clip
	// rectangle. A space clears cells.
	Repeat(x, y, n int, cluster string)
}

// CellNode is an optional interface for a Node that can draw straight into a
// cell grid instead of building a string. DrawCells draws the node as
// Render(Size{r.W, r.H}) would, with its top-left corner at r.X, r.Y of dst,
// clipped to r and to dst's clip, and leaves dst's clip as it found it. It
// writes only the cells it has ink for: the cells of r are expected to be
// blank (DrawTo clears the root's rectangle first), so a CellNode need not
// paint padding. Row, Column, GridNode, BoxNode, Scroll, OverlayNode, Block,
// Text and StyledText implement it; a container draws a child that does not
// implement it by rendering it to a string, so any tree can be drawn.
type CellNode interface {
	Node
	DrawCells(dst CellSurface, r Rect)
}

// DrawTo is Draw into a cell grid: it measures root under c, clears that many
// cells at dst's origin and draws root into them, returning the size drawn
// (the zero Size if either dimension is 0). The screen it produces is the one
// Draw's string would parse to. Like Draw it measures each built-in
// container's children once per distinct Constraints, and for a tree of
// CellNodes it allocates nothing per frame once warm.
func DrawTo(root Node, dst CellSurface, c Constraints) Size {
	a := getArena()
	root = memoized(root, a)
	s := root.Measure(c)
	if s.W > 0 && s.H > 0 {
		r := Rect{W: s.W, H: s.H}
		prev := pushClip(dst, r)
		clearRect(dst, r)
		drawNode(root, dst, r)
		dst.SetClip(prev)
	} else {
		s = Size{}
	}
	putArena(a)
	return s
}

func intersect(a, b Rect) Rect {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	x1, y1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	if x1 <= x0 || y1 <= y0 {
		return Rect{}
	}
	return Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// pushClip narrows dst's clip to r and returns the previous clip to restore.
func pushClip(dst CellSurface, r Rect) Rect {
	prev := dst.Clip()
	dst.SetClip(intersect(prev, r))
	return prev
}

func clearRect(dst CellSurface, r Rect) {
	for y := r.Y; y < r.Y+r.H; y++ {
		dst.Repeat(r.X, y, r.W, " ")
	}
}

// drawNode draws n into r: directly when n is a CellNode, otherwise by
// rendering it to a string and writing its rows.
func drawNode(n Node, dst CellSurface, r Rect) {
	if n == nil || r.Empty() {
		return
	}
	if cn, ok := n.(CellNode); ok {
		cn.DrawCells(dst, r)
		return
	}
	drawRendered(n, dst, r)
}

func drawRendered(n Node, dst CellSurface, r Rect) {
	prev := pushClip(dst, r)
	out := n.Render(Size{W: r.W, H: r.H})
	y := r.Y
	for rest, more := out, out != ""; more && y < r.Y+r.H; y++ {
		var l string
		l, rest, more = nextLine(rest)
		dst.Put(r.X, y, l)
	}
	dst.SetClip(prev)
}

// plainText reports whether s has no control byte other than newline, so its
// rows can be written as they are.
func plainText(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; (c < 0x20 && c != '\n') || c == 0x7f {
			return false
		}
	}
	return true
}

func (b block) DrawCells(dst CellSurface, r Rect) {
	if r.Empty() || b == "" {
		return
	}
	if !plainText(string(b)) {
		drawRendered(b, dst, r)
		return
	}
	prev := pushClip(dst, r)
	y := r.Y
	for rest, more := string(b), true; more && y < r.Y+r.H; y++ {
		var l string
		l, rest, more = nextLine(rest)
		dst.Put(r.X, y, l)
	}
	dst.SetClip(prev)
}

// drawLines writes the styled lines of a text node at its alignment.
func drawTextLines(dst CellSurface, r Rect, lines []string, style ansi.Style, align Align, ellipsis bool, glyph string) {
	prev := pushClip(dst, r)
	for i, l := range lines {
		if i >= r.H {
			break
		}
		if ellipsis && ansi.Width(l) > r.W {
			l = ansi.Truncate(l, r.W-ansi.Width(glyph)) + glyph
		}
		pad := 0
		switch align {
		case AlignCenter:
			pad = (r.W - ansi.Width(l)) / 2
		case AlignEnd:
			pad = r.W - ansi.Width(l)
		}
		if l != "" {
			dst.Put(r.X+max(pad, 0), r.Y+i, style.Render(l))
		}
	}
	dst.SetClip(prev)
}

func (n textNode) DrawCells(dst CellSurface, r Rect) {
	if r.Empty() || n.text == "" {
		return
	}
	drawTextLines(dst, r, n.lines(r.W), n.style, n.align, n.ellipsis, n.glyph)
}

func (n styledTextNode) DrawCells(dst CellSurface, r Rect) {
	if r.Empty() || n.text == "" {
		return
	}
	drawTextLines(dst, r, n.lines(r.W), n.style, AlignStart, false, "")
}

func (f flexNode) DrawCells(dst CellSurface, r Rect) {
	if r.Empty() {
		return
	}
	pl := f.plan(Size{W: r.W, H: r.H})
	prev := pushClip(dst, r)
	pos := pl.lead
	for i, k := range f.kids {
		main := pl.sizes[i]
		off, cs := f.crossPlacement(k, main, pl.crossSize)
		if f.row {
			drawNode(k.Node, dst, Rect{X: r.X + pos, Y: r.Y + off, W: main, H: cs})
		} else {
			drawNode(k.Node, dst, Rect{X: r.X + off, Y: r.Y + pos, W: cs, H: main})
		}
		pos += main + pl.gap
	}
	dst.SetClip(prev)
}

func (g gridNode) DrawCells(dst CellSurface, r Rect) {
	if r.Empty() || len(g.tracks) == 0 || len(g.cells) == 0 {
		return
	}
	widths, heights := g.sizes(Size{W: r.W, H: r.H})
	prev := pushClip(dst, r)
	rowGap, colGap := max(g.rowGap, 0), max(g.colGap, 0)
	y := r.Y
	for row, h := range heights {
		x := r.X
		for c, w := range widths {
			drawNode(g.cell(row, c), dst, Rect{X: x, Y: y, W: w, H: h})
			x += w + colGap
		}
		y += h + rowGap
	}
	dst.SetClip(prev)
}

func (b boxNode) DrawCells(dst CellSurface, r Rect) {
	if r.Empty() {
		return
	}
	bx := b.box
	ch := b.chrome()
	inner := Size{W: r.W - ch.W, H: r.H - ch.H}
	bordered := hasBorder(bx.border)
	simple := inner.W > 0 && inner.H > 0 && bx.bg == nil && bx.height == 0 &&
		bx.marginT == 0 && bx.marginR == 0 && bx.marginB == 0 && bx.marginL == 0 &&
		(!bordered || (bx.border.Top != "" && bx.border.Bottom != ""))
	if !simple {
		drawRendered(b, dst, r)
		return
	}
	prev := pushClip(dst, r)
	showTop, showBottom := bordered && !bx.noTop, bordered && !bx.noBottom
	showLeft, showRight := bordered && !bx.noLeft, bordered && !bx.noRight
	innerW := inner.W + bx.padLeft + bx.padRight
	lw := 0
	if showLeft {
		lw = ansi.Width(bx.border.Left)
	}
	y := r.Y
	if showTop {
		bx.drawEdge(dst, r.X, y, innerW, showLeft, showRight, bx.border.TopLeft, bx.border.Top, bx.border.TopRight, bx.title)
		y++
	}
	rows := bx.padTop + inner.H + bx.padBottom
	if showLeft || showRight {
		left, right := bx.paint(bx.border.Left), bx.paint(bx.border.Right)
		rx := r.X + lw + innerW
		for i := 0; i < rows; i++ {
			if showLeft {
				dst.Put(r.X, y+i, left)
			}
			if showRight {
				dst.Put(rx, y+i, right)
			}
		}
	}
	drawNode(b.child, dst, Rect{X: r.X + lw + bx.padLeft, Y: y + bx.padTop, W: inner.W, H: inner.H})
	y += rows
	if showBottom {
		bx.drawEdge(dst, r.X, y, innerW, showLeft, showRight, bx.border.BottomLeft, bx.border.Bottom, bx.border.BottomRight, "")
	}
	dst.SetClip(prev)
}

// paint colours a border glyph with the box's border colour, if it has one.
func (b Box) paint(s string) string {
	if b.borderColor == nil || s == "" {
		return s
	}
	return ansi.NewStyle().Foreground(b.borderColor).Render(s)
}

// drawEdge draws one horizontal border row (corners and edge, with title
// embedded when set) as Render does, innerW edge cells wide.
func (b Box) drawEdge(dst CellSurface, x, y, innerW int, showLeft, showRight bool, tl, line, tr, title string) {
	if showLeft && tl != "" {
		x += dst.Put(x, y, b.paint(tl))
	}
	if title == "" {
		dst.Repeat(x, y, innerW, b.paint(line))
		x += innerW * ansi.Width(line)
	} else {
		e := b.edge(line, innerW, title)
		dst.Put(x, y, b.paint(e))
		x += ansi.Width(e)
	}
	if showRight && tr != "" {
		dst.Put(x, y, b.paint(tr))
	}
}

func (n scrollNode) DrawCells(dst CellSurface, r Rect) {
	if r.Empty() {
		return
	}
	if _, ok := n.child.(Windowed); ok {
		drawRendered(n, dst, r)
		return
	}
	h := max(n.child.Measure(Constraints{MaxW: r.W, MaxH: Unbounded}).H, r.H)
	off := min(n.offset, h-r.H)
	off = max(off, 0)
	prev := pushClip(dst, r)
	drawNode(n.child, dst, Rect{X: r.X, Y: r.Y - off, W: r.W, H: h})
	if ps := n.pins(r.W, h); ps != nil {
		top := 0
		for _, p := range ps {
			if p.y-off >= top {
				continue
			}
			pr := Rect{X: r.X, Y: r.Y + top, W: r.W, H: min(p.h, r.H-top)}
			if pr.Empty() {
				break
			}
			inner := pushClip(dst, pr)
			clearRect(dst, pr)
			drawNode(p.node, dst, Rect{X: r.X, Y: pr.Y, W: r.W, H: p.h})
			dst.SetClip(inner)
			top += p.h
		}
	}
	dst.SetClip(prev)
}

// OverlayNode composites over on top of base at column x, row y (relative to
// base's top-left; negative values count as 0), replacing what is under it.
// It is Overlay for nodes: Measure is base's, and over is drawn at its own
// natural size, cut off at base's allotted Size. Where Overlay works on
// rendered strings, OverlayNode implements CellNode, so a modal or toast
// layers over a cell-drawn tree without rendering either to a string.
func OverlayNode(base, over Node, x, y int) Node {
	return overlayNode{base: base, over: over, x: max(x, 0), y: max(y, 0)}
}

type overlayNode struct {
	base, over Node
	x, y       int
}

func (o overlayNode) Measure(c Constraints) Size { return o.base.Measure(c) }

// size is over's natural size when placed at o's offset inside s.
func (o overlayNode) size(s Size) Size {
	return o.over.Measure(Constraints{MaxW: max(s.W-o.x, 0), MaxH: max(s.H-o.y, 0)})
}

func (o overlayNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	base := o.base.Render(s)
	if os := o.size(s); os.W > 0 && os.H > 0 {
		base = Overlay(base, o.over.Render(os), o.x, o.y)
	}
	return Block(base).Render(s)
}

func (o overlayNode) DrawCells(dst CellSurface, r Rect) {
	if r.Empty() {
		return
	}
	prev := pushClip(dst, r)
	drawNode(o.base, dst, r)
	if os := o.size(Size{W: r.W, H: r.H}); os.W > 0 && os.H > 0 {
		or := Rect{X: r.X + o.x, Y: r.Y + o.y, W: os.W, H: os.H}
		clearRect(dst, or)
		drawNode(o.over, dst, or)
	}
	dst.SetClip(prev)
}

var (
	_ CellNode = (*memoNode)(nil)
	_ CellNode = flexNode{}
	_ CellNode = gridNode{}
	_ CellNode = boxNode{}
	_ CellNode = scrollNode{}
	_ CellNode = overlayNode{}
	_ CellNode = block("")
	_ CellNode = textNode{}
	_ CellNode = styledTextNode{}
)
