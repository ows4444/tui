package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Panel renders content in a padded, bordered container like Box, but
// with its title on its own reverse-styled row, separated from the body
// by a divider line, rather than just a bold line — Box for when the
// title needs to stand apart from the content, not just be labeled.
// width behaves exactly as it does for Box.
func Panel(title, content string, t theme.Theme, width int) string {
	return panel(title, content, t, width, t.Border)
}

// panel is Panel's implementation, parameterized on the border style so
// Card can reuse it with a forced layout.RoundedBorder().
func panel(title, content string, t theme.Theme, width int, border layout.Border) string {
	cw, auto := boxContentWidth(width)

	wrapped := content
	if !auto {
		wrapped = ansi.WrapStyled(content, cw)
	}

	if title == "" {
		b := layout.NewBox().Border(border).BorderColor(t.BorderColor).PaddingAll(1)
		if !auto {
			b = b.Width(cw)
		}
		return b.Render(wrapped)
	}

	if auto {
		cw = ansi.Width(title)
		for _, l := range strings.Split(wrapped, "\n") {
			if w := ansi.Width(l); w > cw {
				cw = w
			}
		}
	}

	titleBar := ansi.NewStyle().Bold().Background(t.Primary).Foreground(t.TextInverse).Render(padRow(title, cw))
	divider := ansi.NewStyle().Foreground(t.BorderColor).Render(strings.Repeat(t.GlyphSet().RuleH, cw))
	body := titleBar + "\n" + divider
	if wrapped != "" {
		body += "\n" + wrapped
	}

	b := layout.NewBox().Border(border).BorderColor(t.BorderColor).PaddingAll(1).Width(cw)
	return b.Render(body)
}
