package widgets

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// StatusIndicator renders a colored dot followed by a label, e.g. a green
// "● Running", colored by variant via t's matching semantic color.
func StatusIndicator(label string, variant Variant, t theme.Theme) string {
	glyph := t.GlyphSet().Dot
	if m := variant.Mark(t.GlyphSet()); m != "" {
		glyph = m // the variant shows without colour too: its icon replaces the dot
	}
	return ansi.NewStyle().Foreground(variant.Color(t)).Render(glyph) + " " + label
}
