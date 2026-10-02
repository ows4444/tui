package drawer

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode returns the drawer's bordered box as an ordinary layout.Node,
// sized by the layout it is placed in rather than by Width, Height or Edge.
// While the drawer is closed it measures as nothing and renders blank. The
// Model is not changed.
func (m Model) LayoutNode() layout.Node {
	if !m.open {
		return layout.Block("")
	}
	return layout.BoxNode(layout.NewBox().Border(m.themed().Border).BorderColor(m.themed().BorderColor).PaddingAll(1), layout.StyledText(m.Content, ansi.Style{}, false))
}
