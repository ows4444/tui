package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
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

	widths, total := columnWidths(headers, rows)

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

// columnWidths returns the width of each column, the widest of its header and
// cells, and the width of the whole table with two spaces between columns.
func columnWidths(headers []string, rows [][]string) (widths []int, total int) {
	widths = make([]int, len(headers))
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
	for i, w := range widths {
		if i > 0 {
			total += 2
		}
		total += w
	}
	return widths, total
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

// TableOptions shapes a table drawn by TableCells.
type TableOptions struct {
	// Align places the text of each column within its width, and its header:
	// layout.AlignStart (the default) flush left, AlignEnd flush right (numbers),
	// AlignCenter centred. A column past the end of the slice is flush left.
	Align []layout.Align
	// RowStyle styles every cell of data row i; nil leaves rows unstyled.
	RowStyle func(i int) ansi.Style
	// CellStyle styles the cell at data row i, column j, over RowStyle: a cell
	// whose style is the zero ansi.Style{} takes its row's. A status column or a
	// negative number is coloured with it. nil styles no cell.
	CellStyle func(i, j int) ansi.Style
}

// TableCells renders headers and rows like Table (the same header, divider and
// column widths, with t's colours and rule glyph) with a column alignment and
// a style for each row and each cell from opts. Table, which has neither,
// is TableCells with the zero TableOptions.
//
//	widgets.TableCells([]string{"Job", "Status", "Time"}, rows, widgets.TableOptions{
//		Align: []layout.Align{layout.AlignStart, layout.AlignStart, layout.AlignEnd},
//		CellStyle: func(i, j int) ansi.Style {
//			if j == 1 && rows[i][1] == "failed" {
//				return ansi.NewStyle().Foreground(t.Error)
//			}
//			return ansi.Style{}
//		},
//	}, t)
//
// A cell is measured with escape sequences ignored, so a cell may carry its own
// styling as in Table.
func TableCells(headers []string, rows [][]string, opts TableOptions, t theme.Theme) string {
	if len(headers) == 0 {
		return ""
	}
	widths, total := columnWidths(headers, rows)
	headerStyle := ansi.NewStyle().Bold().Foreground(t.Primary)
	dividerStyle := ansi.NewStyle().Foreground(t.Muted)

	var b strings.Builder
	b.Grow((len(rows)+2)*(total+1) + 3*total + 40*len(widths))
	writeCells(&b, headers, widths, opts.Align, func(int) ansi.Style { return headerStyle })
	b.WriteByte('\n')
	b.WriteString(dividerStyle.Render(strings.Repeat(t.GlyphSet().RuleH, total)))
	for i, row := range rows {
		b.WriteByte('\n')
		var rowStyle ansi.Style
		if opts.RowStyle != nil {
			rowStyle = opts.RowStyle(i)
		}
		writeCells(&b, row, widths, opts.Align, func(j int) ansi.Style {
			if opts.CellStyle != nil {
				if st := opts.CellStyle(i, j); st != (ansi.Style{}) {
					return st
				}
			}
			return rowStyle
		})
	}
	return b.String()
}

// writeCells writes one row, each cell aligned in its column and styled by
// styleOf(column).
func writeCells(b *strings.Builder, cells []string, widths []int, align []layout.Align, styleOf func(col int) ansi.Style) {
	for j, w := range widths {
		if j > 0 {
			b.WriteString("  ")
		}
		cell := ""
		if j < len(cells) {
			cell = cells[j]
		}
		a := layout.AlignStart
		if j < len(align) {
			a = align[j]
		}
		b.WriteString(styleOf(j).Render(alignCell(cell, w, a)))
	}
}

// alignCell pads s to width columns with a.
func alignCell(s string, width int, a layout.Align) string {
	pad := width - ansi.Width(s)
	if pad <= 0 {
		return s
	}
	switch a {
	case layout.AlignEnd:
		return strings.Repeat(" ", pad) + s
	case layout.AlignCenter:
		left := pad / 2
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", pad-left)
	}
	return s + strings.Repeat(" ", pad)
}
