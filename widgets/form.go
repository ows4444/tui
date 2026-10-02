package widgets

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// formFieldGap is the vertical separation Form inserts between each
// FormField-rendered block, per layout.JoinVertical's gap parameter.
const formFieldGap = 1

// FormField renders label above fieldView, styled distinctly from plain
// body text using t.Muted, so a form's labels read as labels rather than
// content. A non-empty err renders as an additional line below fieldView,
// styled in t.Error so it stands apart from both the label and the field;
// an empty err renders no error line at all — not even a blank one, so a
// field without a validation error doesn't shift the layout below it.
// FormField does not bound or truncate its output to any width: it is a
// composition helper, not a fixed-width container like widgets.Box.
func FormField(label string, fieldView string, err string, t theme.Theme) string {
	labelLine := ansi.NewStyle().Foreground(t.Muted).Render(label)
	out := labelLine + "\n" + fieldView
	if err != "" {
		errLine := ansi.NewStyle().Foreground(t.Error).Render(err)
		out += "\n" + errLine
	}
	return out
}

// Form stacks multiple already-rendered FormField blocks vertically with a
// blank line of separation between each, as a layout.Column rather than
// re-implementing block-joining. Like FormField, it does not bound or
// truncate its output to any width.
func Form(fields ...string) string {
	return stackBlocks(formFieldGap, fields...)
}

// stackBlocks stacks blocks top to bottom with gap blank rows between them,
// every row padded to the widest block, so the result is a rectangle.
func stackBlocks(gap int, blocks ...string) string {
	kids := make([]layout.FlexChild, len(blocks))
	for i, b := range blocks {
		kids[i] = layout.FlexChild{Node: layout.Block(b)}
	}
	return layout.Draw(layout.Column(gap, kids...), layout.Unconstrained())
}
