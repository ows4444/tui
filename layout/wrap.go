package layout

import "strings"

// Wrap lays children out left to right at their natural width, like words in a
// paragraph, and starts a new row when the next child does not fit in what is
// left of the width. colGap blank columns separate the children of a row and
// rowGap blank rows separate the rows; each row is as tall as its tallest child
// and children are top-aligned. It is for chips, tags, buttons and toolbars that
// should flow instead of being cut: Row cannot wrap. A child wider than the whole
// width sits alone on its row, cut to the width. With no width limit (an
// unbounded Measure) everything is one row. A negative gap counts as 0.
func Wrap(colGap, rowGap int, children ...Node) Node {
	return wrapNode{colGap: max(colGap, 0), rowGap: max(rowGap, 0), kids: children}
}

type wrapNode struct {
	colGap, rowGap int
	kids           []Node
}

// flowKid is a child with the size it is given and where it goes.
type flowKid struct {
	node Node
	rect Rect
}

// flow places the children in width, returning them with their rectangles and
// the size of the whole.
func (w wrapNode) flow(width int) ([]flowKid, Size) {
	var out []flowKid
	x, y, rowH, maxW := 0, 0, 0, 0
	open := Constraints{MaxW: Unbounded, MaxH: Unbounded}
	for _, k := range w.kids {
		sz := k.Measure(open)
		if sz.W <= 0 || sz.H <= 0 {
			continue
		}
		if width < Unbounded && x > 0 && x+sz.W > width {
			y += rowH + w.rowGap
			x, rowH = 0, 0
		}
		cw := sz.W
		if width < Unbounded {
			cw = min(cw, width)
		}
		out = append(out, flowKid{node: k, rect: Rect{X: x, Y: y, W: cw, H: sz.H}})
		x += cw + w.colGap
		rowH = max(rowH, sz.H)
		maxW = max(maxW, x-w.colGap)
	}
	if len(out) == 0 {
		return nil, Size{}
	}
	return out, Size{W: maxW, H: y + rowH}
}

func (w wrapNode) Measure(c Constraints) Size {
	_, sz := w.flow(c.MaxW)
	return c.Constrain(sz)
}

func (w wrapNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	kids, _ := w.flow(s.W)
	if len(kids) == 0 {
		return Block("").Render(s)
	}
	var rows []string
	var heights []int
	for i := 0; i < len(kids); {
		j, h := i, 0
		for j < len(kids) && kids[j].rect.Y == kids[i].rect.Y {
			h = max(h, kids[j].rect.H)
			j++
		}
		blocks := make([]string, 0, j-i)
		for _, k := range kids[i:j] {
			blocks = append(blocks, Block(k.node.Render(Size{W: k.rect.W, H: k.rect.H})).Render(Size{W: k.rect.W, H: h}))
		}
		rows = append(rows, joinHorizontal(w.colGap, blocks...))
		heights = append(heights, h)
		i = j
	}
	return Block(strings.TrimRight(joinColumn(w.rowGap, rows, heights), "\n")).Render(s)
}

func (w wrapNode) DrawCells(dst CellSurface, r Rect) {
	if r.Empty() {
		return
	}
	prev := pushClip(dst, r)
	kids, _ := w.flow(r.W)
	for _, k := range kids {
		drawNode(k.node, dst, Rect{X: r.X + k.rect.X, Y: r.Y + k.rect.Y, W: k.rect.W, H: k.rect.H})
	}
	dst.SetClip(prev)
}

func (w wrapNode) children(s Size) []child {
	kids, _ := w.flow(s.W)
	out := make([]child, len(kids))
	for i, k := range kids {
		out[i] = child{node: k.node, rect: k.rect}
	}
	return out
}
