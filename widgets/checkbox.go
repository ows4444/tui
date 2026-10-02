package widgets

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Checkbox renders a single checked/unchecked indicator ("[x]"/"[ ]")
// followed by label — a standalone InkUI-style Checkbox. focused=true
// colors the indicator with t.Focus so it reads as visually distinct from
// an unfocused one; checked=true additionally colors the indicator with
// t.Success (bold) so checked/unchecked stay distinct from each other
// regardless of focus. An empty label renders just the indicator, with no
// trailing space.
//
// For a list of checkboxes, see multiselect.Model (InkUI's CheckboxGroup):
// it already implements cursor navigation plus per-item Space-toggle, so
// this function doesn't duplicate that as a separate group widget.
func Checkbox(label string, checked, focused bool, t theme.Theme) string {
	mark := " "
	if checked {
		mark = "x"
	}
	indicator := "[" + mark + "]"

	style := ansi.NewStyle()
	switch {
	case focused && checked:
		style = style.Foreground(t.Success).Bold()
	case focused:
		style = t.ResolvedStates().Focus.Bold()
	case checked:
		style = style.Foreground(t.Success)
	default:
		style = style.Foreground(t.Muted)
	}
	rendered := style.Render(indicator)

	if label == "" {
		return rendered
	}
	return rendered + " " + label
}
