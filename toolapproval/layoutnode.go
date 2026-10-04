package toolapproval

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the prompt to a layout.Node: the description and the
// body word-wrap to the allotted width, and the rows are cut to the width and height. The
// Model is not changed.
func (m Model) LayoutNode() layout.Node { return promptNode{m} }

type promptNode struct{ m Model }

func (n promptNode) wrapped(width int) Model {
	m := n.m
	if width > 0 && width < layout.Unbounded {
		m.Description = ansi.WrapStyled(m.Description, width)
		m.Body = ansi.WrapStyled(m.Body, width)
	}
	return m
}

func (n promptNode) Measure(c layout.Constraints) layout.Size {
	return layout.Block(n.wrapped(c.MaxW).View()).Measure(c)
}

func (n promptNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	return layout.Block(n.wrapped(s.W).View()).Render(s)
}
