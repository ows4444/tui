package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Stepper renders a horizontal sequence of step labels, marking steps
// before current as done, current as active, and the rest as upcoming.
// Like ProgressBar, it's stateless — the caller owns "which step we're
// on" and passes it in fresh each render.
func Stepper(steps []string, current int, t theme.Theme) string {
	done := ansi.NewStyle().Foreground(t.Success)
	active := ansi.NewStyle().Bold().Foreground(t.Primary)
	upcoming := ansi.NewStyle().Foreground(t.Muted)
	g := t.GlyphSet()
	sep := upcoming.Render(" " + g.Arrow + " ")

	parts := make([]string, len(steps))
	for i, s := range steps {
		switch {
		case i < current:
			parts[i] = done.Render(g.Check + " " + s)
		case i == current:
			parts[i] = active.Render(g.Dot + " " + s)
		default:
			parts[i] = upcoming.Render(g.DotEmpty + " " + s)
		}
	}
	return strings.Join(parts, sep)
}
