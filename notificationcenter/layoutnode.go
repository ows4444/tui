package notificationcenter

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode returns the notification panel as an ordinary layout.Node: each
// message word-wrapped to the width the panel is given and coloured by its
// Variant, inside a bordered box. An empty queue measures as nothing and
// renders blank. The Model is not changed.
func (m Model) LayoutNode() layout.Node {
	if len(m.Notifications) == 0 {
		return layout.Block("")
	}
	rows := make([]layout.FlexChild, len(m.Notifications))
	for i, n := range m.Notifications {
		style := ansi.NewStyle().Foreground(n.Variant.Color(m.themed()))
		rows[i] = layout.FlexChild{Node: layout.StyledText(ansi.Clean(m.Raw, n.Message), style, true)}
	}
	return layout.BoxNode(layout.NewBox().Border(m.themed().Border).BorderColor(m.themed().BorderColor).PaddingAll(1), layout.Column(0, rows...))
}
