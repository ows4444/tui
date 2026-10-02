package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Table renders a static, column-aligned table: a styled header row, a
// divider, then each data row. Like ProgressBar, it's stateless — for row
// highlighting/navigation, see the datatable package instead.
func Table(headers []string, rows [][]string, t theme.Theme) string {
	headerStyle := ansi.NewStyle().Bold().Foreground(t.Primary)
	dividerStyle := ansi.NewStyle().Foreground(t.Muted)
	return tableRows(headers, rows, headerStyle, dividerStyle, func(int) ansi.Style { return ansi.Style{} }, t.GlyphSet().RuleH)
}

// TableRows renders headers/rows with the same column-width computation,
// divider, and cell padding as Table, except each data row's style comes
// from styleForRow(i) instead of always being unstyled — the shared
// implementation datatable.Model.View uses to highlight its cursor row
// without reimplementing column layout. headerStyle and dividerStyle style
// the header row and the divider line respectively. An empty headers
// returns "".
func TableRows(headers []string, rows [][]string, headerStyle, dividerStyle ansi.Style, styleForRow func(i int) ansi.Style) string {
	return tableRows(headers, rows, headerStyle, dividerStyle, styleForRow, theme.UnicodeGlyphSet().RuleH)
}

// TableRowsWith is TableRows with the divider drawn from t's rule glyph
// (theme.Glyphs.RuleH), so an ASCII theme gets a row of '-'.
func TableRowsWith(headers []string, rows [][]string, headerStyle, dividerStyle ansi.Style, styleForRow func(i int) ansi.Style, t theme.Theme) string {
	return tableRows(headers, rows, headerStyle, dividerStyle, styleForRow, t.GlyphSet().RuleH)
}

func tableRows(headers []string, rows [][]string, headerStyle, dividerStyle ansi.Style, styleForRow func(i int) ansi.Style, rule string) string {
	if len(headers) == 0 {
		return ""
	}

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = ansi.Width(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if w := ansi.Width(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	total := 0
	for i, w := range widths {
		if i > 0 {
			total += 2
		}
		total += w
	}

	// One allocation for the whole table: the text, a multi-byte divider, and room
	// for the style escapes on the header and one highlighted row.
	var b strings.Builder
	b.Grow((len(rows)+2)*(total+1) + 3*total + 40*len(widths))
	writeRow(&b, headers, widths, headerStyle)
	b.WriteByte('\n')
	b.WriteString(dividerStyle.Render(strings.Repeat(rule, total)))

	for i, row := range rows {
		b.WriteByte('\n')
		writeRow(&b, row, widths, styleForRow(i))
	}
	return b.String()
}

func writeRow(b *strings.Builder, cells []string, widths []int, style ansi.Style) {
	for i, w := range widths {
		if i > 0 {
			b.WriteString("  ")
		}
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		b.WriteString(style.Render(padRight(cell, w)))
	}
}

func padRight(s string, width int) string {
	pad := width - ansi.Width(s)
	if pad <= 0 {
		return s
	}
	return s + strings.Repeat(" ", pad)
}
