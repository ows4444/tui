package textinput

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the input to a layout.Node. Measure reports its natural
// size: the prompt, the value (or placeholder) and one cell for the cursor, on
// one row. Render fits the allotted Size by using the width left after the
// prompt as the visible text window (one cell is kept for the cursor), so a
// long value scrolls around the cursor as View does when Width is set. The
// Model is not changed.
//
// Widgets that embed Model and hold a secret must override this too (see
// passwordinput and maskedinput), or the promoted method would draw the real
// value.
func (m Model) LayoutNode() layout.Node { return inputNode{m} }

type inputNode struct{ m Model }

func (n inputNode) Measure(c layout.Constraints) layout.Size {
	text := ansi.Width(n.m.Placeholder)
	if n.m.ed.Len() > 0 {
		text = n.m.ed.Width()
	}
	return c.Constrain(layout.Size{W: ansi.Width(n.m.Prompt) + text + 1, H: 1})
}

func (n inputNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	m.Width = max(s.W-ansi.Width(m.Prompt)-1, 1)
	return layout.Block(m.View()).Render(s)
}
