package helpscreen

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/widgets"
)

// LayoutNode returns the help box as an ordinary layout.Node: one row per key
// binding inside a bordered box, cut to the size it is given. While the screen
// is closed it measures as nothing and renders blank. The Model is not
// changed.
func (m Model) LayoutNode() layout.Node {
	if !m.open {
		return layout.Block("")
	}
	rows := make([]layout.FlexChild, len(m.Hints))
	for i, h := range m.Hints {
		rows[i] = layout.FlexChild{Node: layout.Block(m.hint(h))}
	}
	return layout.BoxNode(layout.NewBox().Border(m.themed().Border).BorderColor(m.themed().BorderColor).PaddingAll(1), layout.Column(0, rows...))
}

// hint renders one "[key] action" row like widgets.KeyHint, with the key in
// the theme's Typography.Strong.
func (m Model) hint(h widgets.Hint) string {
	dim := ansi.NewStyle().Faint()
	return dim.Render("[") + m.themed().ResolvedTypography().Strong.Render(h.Key) + dim.Render("] "+h.Action)
}
