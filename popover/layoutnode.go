package popover

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode returns the popover's bordered box as an ordinary layout.Node
// (Content unwrapped, as Render draws it), cut to the size it is given. While
// the popover is closed it measures as nothing and renders blank. The Model
// is not changed.
func (m Model) LayoutNode() layout.Node {
	if !m.open {
		return layout.Block("")
	}
	return layout.BoxNode(layout.NewBox().Border(m.themed().Border).BorderColor(m.themed().BorderColor).PaddingAll(m.themed().ResolvedSpacing().S), layout.StyledText(m.Content, ansi.Style{}, false))
}
