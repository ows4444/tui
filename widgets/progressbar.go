package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// ProgressBar renders a determinate progress meter width columns wide,
// filled proportionally to percent (clamped to [0,1]) — the static
// counterpart to an animated spinner/loading-bar widget. Unlike those, it
// has no internal state: the caller owns "current progress" (bytes
// downloaded / total, steps done / total, ...) and passes it in fresh on
// every render.
func ProgressBar(percent float64, width int, t theme.Theme) string {
	if width <= 0 {
		return ""
	}
	switch {
	case percent < 0:
		percent = 0
	case percent > 1:
		percent = 1
	}

	filled := int(float64(width)*percent + 0.5)
	if filled > width {
		filled = width
	}

	fill := ansi.NewStyle().Foreground(t.Primary).Render(strings.Repeat(t.GlyphSet().BarFull, filled))
	track := ansi.NewStyle().Foreground(t.Muted).Render(strings.Repeat(t.GlyphSet().BarEmpty, width-filled))
	return fill + track
}
