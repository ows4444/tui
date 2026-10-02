package scrollbar

import "github.com/ows4444/tui/layout"

// LayoutNode adapts the scrollbar to a layout.Node (also a layout.CellNode).
// Measure reports one cell across and Length along the bar's axis; Render and
// DrawCells fit the allotted Size, so the track is as long as the space the
// layout gives it (Length and Bounds are ignored there). The Model is not
// changed.
func (m Model) LayoutNode() layout.Node { return barNode{m} }

type barNode struct{ m Model }

func (n barNode) Measure(c layout.Constraints) layout.Size {
	if n.m.Orientation == Horizontal {
		return c.Constrain(layout.Size{W: max(n.m.Length, 0), H: 1})
	}
	return c.Constrain(layout.Size{W: 1, H: max(n.m.Length, 0)})
}

func (n barNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	return layout.Block(n.fitted(s)).Render(s)
}

// fitted renders the bar along the long axis of s, one cell across.
func (n barNode) fitted(s layout.Size) string {
	if n.m.Orientation == Horizontal {
		return n.m.render(s.W)
	}
	return n.m.render(s.H)
}

// DrawCells draws the bar into r of dst, the screen Render shows.
func (n barNode) DrawCells(dst layout.CellSurface, r layout.Rect) {
	if r.Empty() {
		return
	}
	prev := dst.Clip()
	dst.SetClip(clip(prev, r))
	if n.m.Orientation == Horizontal {
		for i, c := range n.m.cells(r.W) {
			dst.Put(r.X+i, r.Y, c)
		}
	} else {
		for i, c := range n.m.cells(r.H) {
			dst.Put(r.X, r.Y+i, c)
		}
	}
	dst.SetClip(prev)
}

func clip(a, b layout.Rect) layout.Rect {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	x1, y1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	if x1 <= x0 || y1 <= y0 {
		return layout.Rect{}
	}
	return layout.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

var _ layout.CellNode = barNode{}
