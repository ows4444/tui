package inputgroup

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the group to a layout.Node on one row. It measures as
// the field does plus the two additions; given a width, the additions keep
// theirs and the field takes what is left, so Suffix ends at the right edge.
// The Model is not changed.
func (m Model) LayoutNode() layout.Node { return groupNode{m} }

type groupNode struct{ m Model }

func (n groupNode) extra() int { return ansi.Width(n.m.prefix()) + ansi.Width(n.m.suffix()) }

func (n groupNode) Measure(c layout.Constraints) layout.Size {
	in := n.m.Model.LayoutNode().Measure(layout.Constraints{MaxW: layout.Unbounded, MaxH: layout.Unbounded})
	return c.Constrain(layout.Size{W: in.W + n.extra(), H: 1})
}

func (n groupNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	m.Width = max(s.W-n.extra()-ansi.Width(m.Prompt)-1, 1)
	return layout.Block(m.View()).Render(s)
}
