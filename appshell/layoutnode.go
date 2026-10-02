package appshell

import (
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/widgets"
)

// LayoutNode adapts the shell to a layout.Node: the title, the input (which
// scrolls to fit) and the key hints keep their natural height, and the content
// fills the space between them. The Model is not changed.
func (m Model) LayoutNode() layout.Node {
	children := []layout.FlexChild{
		{Node: layout.Block(widgets.Header(m.Title, m.themed()))},
		{Node: m.Input.LayoutNode()},
		layout.Fill(m.Content.LayoutNode()),
	}
	if len(m.Hints) > 0 {
		children = append(children, layout.FlexChild{Node: layout.Block(widgets.KeyHints(" ", m.Hints...))})
	}
	return layout.Column(0, children...)
}
