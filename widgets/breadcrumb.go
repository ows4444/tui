package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Breadcrumb renders a horizontal "a / b / c" navigation trail: each item
// separated by a muted " / " separator, with the last item styled
// distinctly (bold, Theme's Primary color) to mark the current location.
// Like Stepper, it's stateless — the caller passes the full path fresh
// each render.
func Breadcrumb(items []string, t theme.Theme) string {
	if len(items) == 0 {
		return ""
	}

	item := ansi.NewStyle().Foreground(t.Text)
	current := ansi.NewStyle().Bold().Foreground(t.Primary)
	sep := ansi.NewStyle().Foreground(t.Muted).Render(" / ")

	parts := make([]string, len(items))
	last := len(items) - 1
	for i, s := range items {
		if i == last {
			parts[i] = current.Render(s)
		} else {
			parts[i] = item.Render(s)
		}
	}
	return strings.Join(parts, sep)
}
