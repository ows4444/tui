package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Empty renders the placeholder for a view that has nothing to show: a bold
// title, a description in the muted colour and a hint in the primary colour,
// each centred on its own lines, with a blank row between the title and the
// rest. An empty description or hint is left out. Like the other helpers
// here it is stateless: the caller decides when a list is empty and what to
// say about it.
//
// width is the number of cells to centre in, and the description is wrapped
// to it; with width <= 0 the lines are centred on the widest of them and
// nothing wraps. The three strings are sanitised.
func Empty(title, description, hint string, t theme.Theme, width int) string {
	var lines []string
	add := func(s string, style ansi.Style) {
		s = ansi.Sanitize(s)
		if s == "" {
			return
		}
		if width > 0 {
			s = ansi.Wrap(s, width)
		}
		for _, l := range strings.Split(s, "\n") {
			lines = append(lines, style.Render(strings.TrimSpace(l)))
		}
	}
	add(title, ansi.NewStyle().Bold().Foreground(t.Text))
	if n := len(lines); n > 0 && (ansi.Sanitize(description) != "" || ansi.Sanitize(hint) != "") {
		lines = append(lines, "")
	}
	add(description, ansi.NewStyle().Foreground(t.Muted))
	add(hint, ansi.NewStyle().Foreground(t.Primary))

	if width <= 0 {
		for _, l := range lines {
			width = max(width, ansi.Width(l))
		}
	}
	for i, l := range lines {
		w := ansi.Width(l)
		if w > width {
			l, w = ansi.Truncate(l, width), width
		}
		left := (width - w) / 2
		lines[i] = strings.Repeat(" ", left) + l + strings.Repeat(" ", width-w-left)
	}
	return strings.Join(lines, "\n")
}
