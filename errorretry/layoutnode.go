package errorretry

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the error to a layout.Node: the message and the hint
// beneath it each word-wrap to the allotted width, keeping their styles. The
// Model is not changed.
func (m Model) LayoutNode() layout.Node {
	errStyle := ansi.NewStyle().Foreground(m.themed().Error).Bold()
	hintStyle := ansi.NewStyle().Foreground(m.themed().Muted)
	return layout.Column(0,
		layout.FlexChild{Node: layout.StyledText(m.Message, errStyle, true)},
		layout.FlexChild{Node: layout.StyledText(m.hint(), hintStyle, true)},
	)
}
