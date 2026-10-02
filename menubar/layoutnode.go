package menubar

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode returns the bar as an ordinary layout.Node (a layout.CellNode,
// so it draws straight into a cell grid), cut to the size it is given. The
// first row holds the titles; while a dropdown is open it is drawn below them,
// inside the node's size. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return barNode{m} }

type barNode struct{ m Model }

func (n barNode) Measure(c layout.Constraints) layout.Size {
	w, h := ansi.Width(n.m.View()), 1
	if n.m.open {
		d := n.m.drop.Bounds()
		w, h = max(w, d.X-n.m.Bounds.X+d.W), max(h, d.Y-n.m.Bounds.Y+d.H)
	}
	return c.Constrain(layout.Size{W: w, H: h})
}

func (n barNode) Render(s layout.Size) string { return n.node(s).Render(s) }

// DrawCells implements layout.CellNode.
func (n barNode) DrawCells(dst layout.CellSurface, r layout.Rect) {
	if cn, ok := n.node(layout.Size{W: r.W, H: r.H}).(layout.CellNode); ok {
		cn.DrawCells(dst, r)
	}
}

// node is the bar and open dropdown composed into a block of s.
func (n barNode) node(s layout.Size) layout.Node {
	if s.W <= 0 || s.H <= 0 {
		return layout.Block("")
	}
	m := n.m
	rows := make([]string, s.H)
	for i := range rows {
		rows[i] = strings.Repeat(" ", s.W)
	}
	rows[0] = layout.Block(m.View()).Render(layout.Size{W: s.W, H: 1})
	out := strings.Join(rows, "\n")
	if m.open {
		d := m.drop
		d.AnchorX, d.AnchorY = d.AnchorX-m.Bounds.X, d.AnchorY-m.Bounds.Y // node-local
		d.ScreenW, d.ScreenH = s.W, s.H
		out = d.Render(out)
	}
	return layout.Block(out)
}
