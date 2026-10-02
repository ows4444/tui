package markdown

import (
	"sort"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// tableSepW is the width of the " | " between columns.
const tableSepW = 3

// renderTable lays a table out in at most avail columns: cells are inline
// markdown, the header is bold, columns are padded to their alignment and
// joined by the theme's vertical rule, and a horizontal rule follows the
// header. Columns keep their natural width when the table fits; otherwise
// the widest are narrowed evenly and their cells wrap (a word longer than
// the column is broken by column), and any row still wider than avail, at
// tiny widths, is cut.
func renderTable(tb table, avail int, t theme.Theme) []string {
	n := len(tb.header)
	g := t.GlyphSet()

	cells := func(row []string, base look) [][]string {
		out := make([][]string, n)
		for c, text := range row {
			out[c] = wrapPieces(parseInline(text), 1<<20, base, t)
		}
		return out
	}
	// Natural widths come from unwrapped cells.
	widths := make([]int, n)
	head := cells(tb.header, headingLook(3, t))
	body := make([][][]string, len(tb.rows))
	for r, row := range tb.rows {
		body[r] = cells(row, look{})
	}
	for _, row := range append([][][]string{head}, body...) {
		for c, lines := range row {
			for _, l := range lines {
				widths[c] = max(widths[c], ansi.Width(l))
			}
		}
	}
	widths = fitColumns(widths, avail-tableSepW*(n-1))

	sep := " " + ansi.NewStyle().Foreground(t.Muted).Render(g.RuleV) + " "
	line := func(row []string, base look) []string {
		wrapped := make([][]string, n)
		height := 1
		for c, text := range row {
			wrapped[c] = wrapPieces(parseInline(text), widths[c], base, t)
			height = max(height, len(wrapped[c]))
		}
		out := make([]string, height)
		for h := range out {
			parts := make([]string, n)
			for c := range parts {
				cell := ""
				if h < len(wrapped[c]) {
					cell = wrapped[c][h]
				}
				parts[c] = alignCell(cell, widths[c], tb.align[c])
			}
			out[h] = ansi.Truncate(strings.TrimRight(strings.Join(parts, sep), " "), avail)
		}
		return out
	}

	out := line(tb.header, headingLook(3, t))
	total := tableSepW * (n - 1)
	for _, w := range widths {
		total += w
	}
	rule := ansi.NewStyle().Foreground(t.Muted).Render(strings.Repeat(g.RuleH, min(total, avail)))
	out = append(out, rule)
	for _, row := range tb.rows {
		out = append(out, line(row, look{})...)
	}
	return out
}

// alignCell pads a styled cell to w columns by alignment: 'r' right, 'c'
// centred (the odd column goes right), else left.
func alignCell(cell string, w int, align byte) string {
	pad := max(w-ansi.Width(cell), 0)
	switch align {
	case 'r':
		return strings.Repeat(" ", pad) + cell
	case 'c':
		return strings.Repeat(" ", pad/2) + cell + strings.Repeat(" ", pad-pad/2)
	}
	return cell + strings.Repeat(" ", pad)
}

// fitColumns narrows the widest columns so they total at most budget,
// keeping every column at least one wide. It caps all columns at the
// largest c that fits and hands the leftover columns out to the widest.
func fitColumns(widths []int, budget int) []int {
	sum := 0
	for _, w := range widths {
		sum += w
	}
	if sum <= budget {
		return widths
	}
	total := func(c int) int {
		s := 0
		for _, w := range widths {
			s += min(w, c)
		}
		return s
	}
	c := sort.Search(sum+1, func(c int) bool { return total(c+1) > budget })
	c = max(c, 1)
	out := make([]int, len(widths))
	left := budget - total(c)
	for i, w := range widths {
		out[i] = max(min(w, c), 1)
	}
	for i := range out {
		if left <= 0 {
			break
		}
		if widths[i] > out[i] {
			out[i]++
			left--
		}
	}
	return out
}
