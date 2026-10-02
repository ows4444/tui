package widgets

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Pagination renders a compact "Page X of Y" indicator (1-indexed). current
// is clamped into [1, total] so an out-of-range value never produces a
// nonsensical or panicking result. Like ProgressBar, it's stateless — the
// caller owns "current page" and passes it in fresh on every render.
func Pagination(current, total int, t theme.Theme) string {
	if total <= 0 {
		return ""
	}
	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}
	return fmt.Sprintf("Page %d of %d", current, total)
}

// PaginationDots renders one dot per page, filled for the current page
// (Theme's Primary color) and hollow for the rest (Theme's Muted color) —
// mirroring ProgressBar's filled/track color convention as a compact page
// indicator alternative to Pagination's "Page X of Y" text.
func PaginationDots(current, total int, t theme.Theme) string {
	if total <= 0 {
		return ""
	}
	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}

	g := t.GlyphSet()
	filled := ansi.NewStyle().Foreground(t.Primary)
	track := ansi.NewStyle().Foreground(t.Muted)

	var b strings.Builder
	for i := 1; i <= total; i++ {
		if i == current {
			b.WriteString(filled.Render(g.Dot))
		} else {
			b.WriteString(track.Render(g.DotEmpty))
		}
	}
	return b.String()
}
