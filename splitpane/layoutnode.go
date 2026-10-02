package splitpane

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the split to a layout.Node (also a layout.CellNode).
// Measure asks both panes under the constraints and reports the larger
// cross-axis size and the panes plus divider along the axis; Render and
// DrawCells fit the allotted Size, clamping the divider to the minimums at
// that size (Bounds and Total are ignored there). The Model is not changed.
func (m Model) LayoutNode() layout.Node { return splitNode{m} }

type splitNode struct{ m Model }

func nodeSize(n layout.Node, c layout.Constraints) layout.Size {
	if n == nil {
		return layout.Size{}
	}
	return n.Measure(c)
}

func (n splitNode) Measure(c layout.Constraints) layout.Size {
	a := nodeSize(n.m.First, c)
	b := nodeSize(n.m.Second, c)
	if n.m.Direction == Rows {
		return c.Constrain(layout.Size{W: max(a.W, b.W), H: a.H + 1 + b.H})
	}
	return c.Constrain(layout.Size{W: a.W + 1 + b.W, H: max(a.H, b.H)})
}

// geometry returns the first pane's size, the divider's offset (the same) and
// the second pane's size along the axis, and the cross size.
func (n splitNode) geometry(s layout.Size) (first, second, cross int, rows bool) {
	rows = n.m.Direction == Rows
	main, cross := s.W, s.H
	if rows {
		main, cross = s.H, s.W
	}
	first = n.m.posFor(main)
	return first, max(main-1-first, 0), cross, rows
}

func (n splitNode) divider() string {
	g := n.m.themed().Glyphs.Resolved()
	if n.m.Direction == Rows {
		return g.RuleH
	}
	return g.RuleV
}

func (n splitNode) dividerStyle() ansi.Style {
	if n.m.dragging {
		return ansi.NewStyle().Foreground(n.m.themed().Primary).Bold()
	}
	return ansi.NewStyle().Foreground(n.m.themed().BorderColor)
}

func renderPane(p layout.Node, s layout.Size) string {
	if p == nil || s.W <= 0 || s.H <= 0 {
		return layout.Block("").Render(s)
	}
	return p.Render(s)
}

func (n splitNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	first, second, _, rows := n.geometry(s)
	div := n.dividerStyle().Render(n.divider())
	if rows {
		line := n.dividerStyle().Render(strings.Repeat(n.divider(), s.W))
		parts := []string{}
		if first > 0 {
			parts = append(parts, renderPane(n.m.First, layout.Size{W: s.W, H: first}))
		}
		parts = append(parts, line)
		if second > 0 {
			parts = append(parts, renderPane(n.m.Second, layout.Size{W: s.W, H: second}))
		}
		return strings.Join(parts, "\n")
	}
	a := strings.Split(renderPane(n.m.First, layout.Size{W: first, H: s.H}), "\n")
	b := strings.Split(renderPane(n.m.Second, layout.Size{W: second, H: s.H}), "\n")
	rowsOut := make([]string, s.H)
	for i := range rowsOut {
		rowsOut[i] = at(a, i, first) + div + at(b, i, second)
	}
	return strings.Join(rowsOut, "\n")
}

// at returns row i of lines, padded to w (an empty pane has no rows).
func at(lines []string, i, w int) string {
	if w <= 0 {
		return ""
	}
	if i < len(lines) {
		l := lines[i]
		if d := w - ansi.Width(l); d > 0 {
			l += strings.Repeat(" ", d)
		}
		return l
	}
	return strings.Repeat(" ", w)
}

// DrawCells draws the panes and divider into r of dst, the screen Render
// shows. A pane that is a layout.CellNode draws straight into dst.
func (n splitNode) DrawCells(dst layout.CellSurface, r layout.Rect) {
	if r.Empty() {
		return
	}
	prev := dst.Clip()
	dst.SetClip(clipTo(prev, r))
	first, second, _, rows := n.geometry(layout.Size{W: r.W, H: r.H})
	style := n.dividerStyle()
	if rows {
		drawPane(dst, n.m.First, layout.Rect{X: r.X, Y: r.Y, W: r.W, H: first})
		dst.Repeat(r.X, r.Y+first, r.W, style.Render(n.divider()))
		drawPane(dst, n.m.Second, layout.Rect{X: r.X, Y: r.Y + first + 1, W: r.W, H: second})
	} else {
		drawPane(dst, n.m.First, layout.Rect{X: r.X, Y: r.Y, W: first, H: r.H})
		for y := r.Y; y < r.Y+r.H; y++ {
			dst.Put(r.X+first, y, style.Render(n.divider()))
		}
		drawPane(dst, n.m.Second, layout.Rect{X: r.X + first + 1, Y: r.Y, W: second, H: r.H})
	}
	dst.SetClip(prev)
}

func drawPane(dst layout.CellSurface, p layout.Node, r layout.Rect) {
	if p == nil || r.Empty() {
		return
	}
	if cn, ok := p.(layout.CellNode); ok {
		cn.DrawCells(dst, r)
		return
	}
	prev := dst.Clip()
	dst.SetClip(clipTo(prev, r))
	for i, l := range strings.Split(p.Render(layout.Size{W: r.W, H: r.H}), "\n") {
		if i >= r.H {
			break
		}
		dst.Put(r.X, r.Y+i, l)
	}
	dst.SetClip(prev)
}

func clipTo(a, b layout.Rect) layout.Rect {
	x0, y0 := max(a.X, b.X), max(a.Y, b.Y)
	x1, y1 := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	if x1 <= x0 || y1 <= y0 {
		return layout.Rect{}
	}
	return layout.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

var _ layout.CellNode = splitNode{}
