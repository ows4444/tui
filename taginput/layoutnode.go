package taginput

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/widgets"
)

// LayoutNode adapts the field to a layout.Node. Measure reports its natural
// size (the tags and the input on one row). Render fits the allotted Size:
// the tags keep their room and the input scrolls its value around the cursor
// in whatever width is left, all cut to the width. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return tagsNode{m} }

type tagsNode struct{ m Model }

// tagsWidth is the width of the rendered tags and the space after each.
func (n tagsNode) tagsWidth() int {
	w := 0
	for _, tag := range n.m.Tags {
		w += ansi.Width(widgets.Tag(tag, widgets.TagSolid, n.m.Variant, n.m.themed())) + 1
	}
	return w
}

func (n tagsNode) Measure(c layout.Constraints) layout.Size {
	text := ansi.Width(n.m.Input.Placeholder)
	if v := len([]rune(n.m.Input.Value())); v > 0 {
		text = v
	}
	return c.Constrain(layout.Size{W: n.tagsWidth() + ansi.Width(n.m.Input.Prompt) + text + 1, H: 1})
}

func (n tagsNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	m.Input.Width = max(s.W-n.tagsWidth()-ansi.Width(m.Input.Prompt)-1, 1)
	return layout.Block(strings.TrimRight(m.View(), "\n")).Render(s)
}
