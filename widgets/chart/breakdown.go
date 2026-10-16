package chart

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// breakdownColours returns the colour of the i-th segment: the theme's accent
// roles in turn, then round again.
func breakdownColour(t theme.Theme, i int) ansi.Color {
	roles := []ansi.Color{t.Primary, t.Secondary, t.Success, t.Warning, t.Error, t.Info}
	return roles[i%len(roles)]
}

// breakdownGlyph returns the fill of the i-th segment, from dark to light, so
// segments differ without colour too. The theme's four shade glyphs are used in
// turn.
func breakdownGlyph(g theme.Glyphs, i int) string {
	shades := []rune(g.Shades)
	if len(shades) == 0 {
		return g.BarFull
	}
	return string(shades[len(shades)-1-i%len(shades)])
}

// BreakdownBar renders items as one stacked bar width columns wide, each
// segment as wide as its share of the total, and a legend under it with a
// marker, the label, the value and the percentage of each. Items with a value
// of 0 or less have no segment but keep their legend line. Segments differ by
// colour (the theme's accent roles in turn) and by shade, so they stay apart
// without colour. Widths are rounded so the segments always fill the bar
// exactly. It renders as "" when width is 0 or less or no value is positive.
func BreakdownBar(items []BarItem, width int, t theme.Theme) string {
	total := 0.0
	for _, it := range items {
		if it.Value > 0 && finite(it.Value) {
			total += it.Value
		}
	}
	if width <= 0 || total <= 0 {
		return ""
	}
	g := t.GlyphSet()
	widths := shareWidths(items, total, width)

	var bar strings.Builder
	var legend []string
	for i, it := range items {
		glyph := breakdownGlyph(g, i)
		style := ansi.NewStyle().Foreground(breakdownColour(t, i))
		if widths[i] > 0 {
			bar.WriteString(style.Render(strings.Repeat(glyph, widths[i])))
		}
		pct := 0.0
		if it.Value > 0 && finite(it.Value) {
			pct = it.Value / total * 100
		}
		legend = append(legend, style.Render(glyph)+" "+it.Label+" "+num(it.Value)+" ("+strconv.FormatFloat(pct, 'f', 0, 64)+"%)")
	}
	return bar.String() + "\n" + strings.Join(legend, "\n")
}

// shareWidths splits width into one whole number of columns per item in
// proportion to its value (largest remainder), so the widths sum to width.
func shareWidths(items []BarItem, total float64, width int) []int {
	widths := make([]int, len(items))
	type frac struct {
		i int
		f float64
	}
	var rest []frac
	used := 0
	for i, it := range items {
		if !(it.Value > 0 && finite(it.Value)) {
			continue
		}
		exact := it.Value / total * float64(width)
		widths[i] = int(math.Floor(exact))
		used += widths[i]
		rest = append(rest, frac{i, exact - math.Floor(exact)})
	}
	sort.SliceStable(rest, func(a, b int) bool { return rest[a].f > rest[b].f })
	for k := 0; used < width && len(rest) > 0; k++ {
		widths[rest[k%len(rest)].i]++
		used++
	}
	return widths
}

// LinearizeBreakdownBar gives a BreakdownBar a plain-text summary: each item
// with its value and share, largest first.
func LinearizeBreakdownBar(items []BarItem) string {
	total := 0.0
	for _, it := range items {
		if it.Value > 0 && finite(it.Value) {
			total += it.Value
		}
	}
	if total <= 0 {
		return "Breakdown: no data"
	}
	sorted := append([]BarItem(nil), items...)
	sort.SliceStable(sorted, func(a, b int) bool { return sorted[a].Value > sorted[b].Value })
	parts := make([]string, 0, len(sorted))
	for _, it := range sorted {
		if it.Value > 0 && finite(it.Value) {
			parts = append(parts, it.Label+" "+num(it.Value)+" ("+strconv.FormatFloat(it.Value/total*100, 'f', 0, 64)+"%)")
		}
	}
	return "Breakdown: " + strings.Join(parts, ", ")
}
