package widgets

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// codeBlockMinWidth is the narrowest width that still fits the border (2
// columns), the padding (2) and one column of code; below it CodeBlock
// drops the frame.
const codeBlockMinWidth = 5

// CodeBlock renders code in a bordered box exactly width columns wide
// (border and padding included), one row per source line, optionally with
// a right-aligned line-number gutter.
//
// It is deliberately plain: code is drawn in t.Text with no syntax
// highlighting (see CodeBlockLang for Go, JSON and shell), the frame uses t.Border in
// t.BorderColor, and line numbers use t.Muted. Lines never wrap — one wider
// than the box is cut and ends in a Muted "…" — so the row count always
// equals the number of source lines plus the two border rows. Tabs become
// four spaces, carriage returns are dropped and one trailing newline is
// ignored. If the gutter would leave no room for code it is omitted, and
// below codeBlockMinWidth columns the frame is dropped and each line is
// just truncated to width. A width of zero or less returns "".
func CodeBlock(code string, width int, lineNumbers bool, t theme.Theme) string {
	return renderCode(codeLines(code), nil, width, lineNumbers, t)
}

// renderCode is CodeBlock's engine, shared with DiffView: it frames lines,
// drawing line i in colors[i] (t.Text when colors is nil). Width and
// gutter rules are documented on CodeBlock.
func renderCode(lines []string, colors []ansi.Color, width int, lineNumbers bool, t theme.Theme) string {
	color := func(i int) ansi.Color {
		if colors == nil {
			return t.Text
		}
		return colors[i]
	}
	return renderCodeRows(len(lines), width, lineNumbers, t,
		func(i, w int) string {
			l := ansi.Truncate(lines[i], w)
			if l != "" {
				l = ansi.NewStyle().Foreground(color(i)).Render(l)
			}
			return l
		},
		func(i, w int) string { return fitCodeLine(lines[i], w, color(i), t) })
}

// renderCodeRows lays out n code rows: the frame, the gutter and the width rules
// documented on CodeBlock. narrow(i, w) returns row i for the frameless
// layout below codeBlockMinWidth (at most w columns); fit(i, w) returns row
// i padded or cut to exactly w columns.
func renderCodeRows(n, width int, lineNumbers bool, t theme.Theme, narrow, fit func(i, w int) string) string {
	if width <= 0 {
		return ""
	}

	if width < codeBlockMinWidth {
		rows := make([]string, n)
		for i := range rows {
			rows[i] = narrow(i, width)
		}
		return strings.Join(rows, "\n")
	}

	inner := width - 4
	digits, gutter := 0, 0
	if lineNumbers {
		digits = len(strconv.Itoa(n))
		gutter = digits + 3 // "<n> │ "
		if inner-gutter < 1 {
			lineNumbers, gutter = false, 0
		}
	}

	muted := ansi.NewStyle().Foreground(t.Muted)
	rows := make([]string, n)
	for i := range rows {
		var row string
		if lineNumbers {
			row = muted.Render(fmt.Sprintf("%*d %s ", digits, i+1, t.GlyphSet().RuleV))
		}
		rows[i] = row + fit(i, inner-gutter)
	}
	return frameRows(rows, inner, t)
}

// codeLines normalises code into display lines (see CodeBlock).
func codeLines(code string) []string {
	code = strings.ReplaceAll(code, "\r", "")
	code = strings.ReplaceAll(code, "\t", "    ")
	code = strings.TrimSuffix(code, "\n")
	return strings.Split(code, "\n")
}

// fitCodeLine returns line rendered in color and padded or cut to exactly
// w columns; a cut line ends in a Muted ellipsis.
func fitCodeLine(line string, w int, color ansi.Color, t theme.Theme) string {
	text := ansi.NewStyle().Foreground(color)
	lw := ansi.Width(line)
	if lw <= w {
		if line == "" {
			return strings.Repeat(" ", w)
		}
		return text.Render(line) + strings.Repeat(" ", w-lw)
	}
	cut := ansi.Truncate(line, w-1)
	out := ansi.NewStyle().Foreground(t.Muted).Render(t.GlyphSet().Ellipsis)
	if cut != "" {
		out = text.Render(cut) + out
	}
	return out + strings.Repeat(" ", w-1-ansi.Width(cut))
}

// frameRows draws t.Border in t.BorderColor around rows that are each
// exactly inner columns wide, with one column of padding either side, so
// every output line is inner+4 columns. Border pieces a theme leaves empty
// are drawn as spaces to keep that width.
func frameRows(rows []string, inner int, t theme.Theme) string {
	b := t.Border
	edge := func(s string) string {
		if s == "" {
			return " "
		}
		return s
	}
	bs := ansi.NewStyle().Foreground(t.BorderColor)
	bar := func(fill, l, r string) string {
		return bs.Render(edge(l) + strings.Repeat(edge(fill), inner+2) + edge(r))
	}

	out := make([]string, 0, len(rows)+2)
	out = append(out, bar(b.Top, b.TopLeft, b.TopRight))
	left, right := bs.Render(edge(b.Left)), bs.Render(edge(b.Right))
	for _, r := range rows {
		out = append(out, left+" "+r+" "+right)
	}
	out = append(out, bar(b.Bottom, b.BottomLeft, b.BottomRight))
	return strings.Join(out, "\n")
}
