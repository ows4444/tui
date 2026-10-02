package widgets

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Toggle renders a single on/off switch ("(●)" on, "( )" off) followed by
// label — a standalone InkUI-style Toggle, the switch-shaped sibling of
// Checkbox. Its coloring follows the same convention Checkbox uses: on
// state is colored with t.Success so it reads as distinct from off (which
// uses t.Muted), and focused=true colors and bolds the indicator with
// t.Focus (or keeps t.Success, bolded, when also on) so focus stays
// visually distinct from an unfocused Toggle the same way it does for
// Checkbox. An empty label renders just the indicator, with no trailing
// space.
//
// For a single-choice list, see picker.Model (InkUI's RadioGroup): this
// function only renders one standalone switch, not a group of them.
func Toggle(label string, on, focused bool, t theme.Theme) string {
	indicator := "( )"
	if on {
		indicator = "(" + t.GlyphSet().Dot + ")"
	}

	style := ansi.NewStyle()
	switch {
	case focused && on:
		style = style.Foreground(t.Success).Bold()
	case focused:
		style = t.ResolvedStates().Focus.Bold()
	case on:
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
