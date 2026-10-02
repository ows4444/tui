package dialog

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode returns the dialog's bordered box as an ordinary layout.Node,
// for placing in a layout instead of compositing over a base: the title, and
// the message word-wrapped to the width the box is given. While the dialog is
// closed it measures as nothing and renders blank. The Model is not changed.
func (m Model) LayoutNode() layout.Node {
	if !m.open {
		return layout.Block("")
	}
	title := m.themed().ResolvedTypography().H2
	children := []layout.FlexChild{{Node: layout.StyledText(ansi.Clean(m.Raw, m.Title), title, false)}}
	if m.Message != "" {
		children = append(children, layout.FlexChild{Node: layout.Block("")}, layout.FlexChild{Node: layout.StyledText(ansi.Clean(m.Raw, m.Message), ansi.Style{}, true)})
	}
	return layout.BoxNode(layout.NewBox().Border(m.themed().Border).BorderColor(m.themed().BorderColor).PaddingAll(1), layout.Column(0, children...))
}
