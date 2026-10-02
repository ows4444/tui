package widgets

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Badge renders text as a small solid-pill label colored by variant (via
// t's matching semantic color as the background, with t.TextInverse as the
// foreground), e.g. a bright-green pill reading " OK ". Every variant but
// VariantNeutral leads its text with Variant.Mark, so the variant shows
// without colour too.
func Badge(text string, variant Variant, t theme.Theme) string {
	return ansi.NewStyle().Bold().Background(variant.Color(t)).Foreground(t.TextInverse).Render(" " + markPrefix(variant, t) + text + " ")
}
