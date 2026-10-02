package widgets

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Banner renders message as a full-width solid bar colored by variant via
// variant.Color(t) as the background (with t.TextInverse as the
// foreground) — no border characters at all, visually distinct from
// Alert's bordered box. Given a positive width, the line is padded (or
// truncated) so its ansi.Width equals width exactly. width<=0 sizes the
// bar to its content instead, with no padding, since Banner has no
// fixed-width requirement to satisfy in that case.
func Banner(message string, variant Variant, t theme.Theme, width int) string {
	message = markPrefix(variant, t) + message
	text := message
	if width > 0 {
		text = padRow(message, width)
	}
	return ansi.NewStyle().Background(variant.Color(t)).Foreground(t.TextInverse).Render(text)
}
