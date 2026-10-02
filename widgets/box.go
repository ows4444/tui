package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// boxFrame is the padding+border overhead a Box/Panel/Card adds around its
// content width: 1 column of padding plus 1 border character on each side.
const boxFrame = 4

// boxContentWidth turns a Box/Panel/Card's total requested width into the
// inner content width layout.Box expects, per the layout.Box convention
// that 0 means "size to content" rather than a fixed width. auto reports
// whether the caller asked for size-to-content, since a computed width of
// 0 is otherwise indistinguishable from that.
func boxContentWidth(width int) (cw int, auto bool) {
	if width <= 0 {
		return 0, true
	}
	cw = width - boxFrame
	if cw < 0 {
		cw = 0
	}
	return cw, false
}

// padRow truncates or right-pads s with spaces so ansi.Width(s) == width
// exactly, for rows (a title bar, a divider) that must span the full
// content width themselves rather than relying on layout.Box's own
// per-line padding.
func padRow(s string, width int) string {
	if ansi.Width(s) > width {
		s = ansi.Truncate(s, width)
	}
	if pad := width - ansi.Width(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

// Box renders content in a padded, bordered container using t's border
// style and color. An empty title renders no title line; a non-empty
// title renders as a bold line above the content, inside the border (the
// same composition dialog.Model.Render uses). width is the box's total
// rendered width, borders and padding included; 0 sizes the box to its
// content instead, matching layout.Box's own convention. When width is
// set, content is word-wrapped (and title truncated) to fit it exactly.
func Box(title, content string, t theme.Theme, width int) string {
	cw, auto := boxContentWidth(width)

	body := content
	if !auto {
		body = ansi.WrapStyled(content, cw)
	}
	if title != "" {
		titleText := title
		if !auto {
			titleText = ansi.Truncate(title, cw)
		}
		titleLine := ansi.NewStyle().Bold().Foreground(t.Primary).Render(titleText)
		if body != "" {
			body = titleLine + "\n" + body
		} else {
			body = titleLine
		}
	}

	b := layout.NewBox().Border(t.Border).BorderColor(t.BorderColor).PaddingAll(1)
	if !auto {
		b = b.Width(cw)
	}
	return b.Render(body)
}
