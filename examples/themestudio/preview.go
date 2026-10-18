package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// orElse is c, or fallback when the theme leaves c unset.
func orElse(c, fallback ansi.Color) ansi.Color {
	if c == nil {
		return fallback
	}
	return c
}

// preview draws a sample screen with the edited theme: the colours, glyphs and
// border a program using it would show. h is the number of rows to fill.
func (m model) preview(ui theme.Theme, h int) layout.Node {
	t := m.t
	w := m.rightW()
	bg := orElse(t.Background, t.TextInverse)
	surface, overlay := orElse(t.Surface, bg), orElse(t.Overlay, bg)
	g := t.GlyphSet()
	on := func(fg, back ansi.Color) ansi.Style { return ansi.NewStyle().Foreground(fg).Background(back) }
	// row joins styled pieces and fills the rest of the line with back.
	row := func(back ansi.Color, parts ...string) string {
		s := strings.Join(parts, "")
		if n := ansi.Width(s); n < w {
			s += on(t.Text, back).Render(strings.Repeat(" ", w-n))
		}
		return ansi.Truncate(s, w)
	}
	txt := func(fg, back ansi.Color, s string) string { return on(fg, back).Render(s) }
	bold := func(fg, back ansi.Color, s string) string { return on(fg, back).Bold().Render(s) }

	bar := 18
	done := bar * 6 / 10
	b := t.Border
	inner := 26
	card := []string{
		txt(t.BorderColor, surface, b.TopLeft+strings.Repeat(b.Top, inner)+b.TopRight),
		txt(t.BorderColor, surface, b.Left) + txt(t.Text, surface, pad(" A card on the surface", inner)) + txt(t.BorderColor, surface, b.Right),
		txt(t.BorderColor, surface, b.BottomLeft+strings.Repeat(b.Bottom, inner)+b.BottomRight),
	}
	lines := []string{
		row(bg, bold(t.Primary, bg, " Preview "), txt(t.Muted, bg, g.Middot+" "+m.sp.base)),
		row(bg, txt(t.Text, bg, " Body text in the text colour, "), txt(t.Muted, bg, "and muted for the quiet parts.")),
		row(bg, " ", bold(t.TextInverse, t.Primary, " Primary "), " ", bold(t.TextInverse, t.Secondary, " Secondary "), " ", txt(t.Focus, bg, g.Cursor+" focused")),
		row(bg, " ", txt(t.Success, bg, g.Check+" done"), "  ", txt(t.Warning, bg, g.Warning+" careful"), "  ", txt(t.Error, bg, g.Cross+" failed"), "  ", txt(t.Info, bg, g.Info+" note")),
		row(bg, " ", txt(t.Primary, bg, strings.Repeat(g.BarFull, done)), txt(t.Muted, bg, strings.Repeat(g.BarEmpty, bar-done)), txt(t.Text, bg, " 60%")),
		row(bg, " ", txt(t.Text, bg, g.Expanded+" tree "), txt(t.Muted, bg, g.Bullet+" muted item  "), txt(t.Text, t.Selection, " selected row ")),
		row(bg, " ", bold(t.Primary, bg, "Tab one"), txt(t.Muted, bg, "  Tab two  Tab three")),
	}
	for _, c := range card {
		lines = append(lines, row(bg, " ", c))
	}
	lines = append(lines, row(bg, " ", txt(t.Text, overlay, pad(" Overlay: menu or dialog ", 28))))
	var kids []layout.FlexChild
	for i := 0; i < max(0, h); i++ {
		l := strings.Repeat(" ", w)
		switch {
		case i < len(lines):
			l = lines[i]
		case i == len(lines):
			l = ""
		}
		kids = append(kids, layout.FlexChild{Node: layout.Block(l)})
	}
	if issues := t.Check(4.5); len(lines) < h-1 {
		kids[len(lines)] = layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(contrastLine(issues), w)))}
	}
	return layout.Fixed(layout.Column(0, kids...), layout.Size{W: w, H: max(0, h)})
}

func contrastLine(issues []theme.ContrastIssue) string {
	if len(issues) == 0 {
		return " Contrast: every text and accent colour reaches 4.5:1 on the background"
	}
	var names []string
	for _, i := range issues {
		names = append(names, fmt.Sprintf("%s %.1f", i.Role, i.Ratio))
	}
	return " Below 4.5:1: " + strings.Join(names, ", ")
}
