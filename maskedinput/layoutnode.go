package maskedinput

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the field to a layout.Node, like textinput.Model's, but
// renders the masked View, never the typed value. This override is required:
// Model embeds textinput.Model, whose LayoutNode would otherwise be promoted
// and draw the real value on screen.
func (m Model) LayoutNode() layout.Node { return maskedNode{m} }

type maskedNode struct{ m Model }

func (n maskedNode) Measure(c layout.Constraints) layout.Size {
	text := ansi.Width(n.m.Placeholder)
	if v := len([]rune(n.m.Value())); v > 0 {
		text = v
	}
	return c.Constrain(layout.Size{W: ansi.Width(n.m.Prompt) + text + 1, H: 1})
}

func (n maskedNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	m.Width = max(s.W-ansi.Width(m.Prompt)-1, 1)
	return layout.Block(m.View()).Render(s)
}
