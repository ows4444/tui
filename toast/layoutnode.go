package toast

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode returns the toast's bordered box as an ordinary layout.Node, its
// message word-wrapped to the width it is given and coloured by Variant. While
// the toast is closed it measures as nothing and renders blank. The Model is
// not changed.
func (m Model) LayoutNode() layout.Node {
	if !m.open {
		return layout.Block("")
	}
	style := ansi.NewStyle().Foreground(m.Variant.Color(m.themed()))
	return layout.BoxNode(layout.NewBox().Border(m.themed().Border).BorderColor(m.themed().BorderColor).PaddingAll(m.themed().ResolvedSpacing().S), layout.StyledText(ansi.Clean(m.Raw, m.Message), style, true))
}
