package chart

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// heatShades is the density scale HeatMap picks cells from, low to high,
// the same idea as sparkBars but for a 2D grid instead of a 1D slice.
var heatShades = shadesFor(theme.UnicodeGlyphSet())

// shadesFor is the density scale for g: a blank for the emptiest level, then
// g.Shades from light to dark.
func shadesFor(g theme.Glyphs) []rune { return []rune(" " + g.Shades) }

// HeatMap renders a 2D grid of values as one output row per row of values,
// with each cell replaced by a shading character from heatShades chosen by
// that value's position between the whole grid's own min and max (not a
// per-row min/max), the same normalization convention as Sparkline. A flat
// grid (min == max across every value) renders a uniform mid-level shade
// for every cell, matching Sparkline's flat-case convention. Rows may have
// different lengths; each renders using its own length. An empty grid (no
// rows, or every row with zero columns) renders "".
//
// Denser cells (the top of the scale) are styled with t.Primary; the
// emptiest cells (level 0, a blank space) are left unstyled since color on
// a blank adds nothing.
func HeatMap(values [][]float64, t theme.Theme) string {
	min, max := 0.0, 0.0
	seen := false
	for _, row := range values {
		for _, v := range row {
			if !seen {
				min, max = v, v
				seen = true
				continue
			}
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
	}
	if !seen {
		return ""
	}
	span := max - min

	fill := ansi.NewStyle().Foreground(t.Primary)
	shades := shadesFor(t.GlyphSet())

	lines := make([]string, len(values))
	for r, row := range values {
		var b strings.Builder
		for _, v := range row {
			var level int
			if span == 0 {
				level = len(shades) / 2
			} else {
				frac := (v - min) / span
				level = int(frac*float64(len(shades)-1) + 0.5) // round to nearest
				if level < 0 {
					level = 0
				}
				if level >= len(shades) {
					level = len(shades) - 1
				}
			}
			ch := string(shades[level])
			if level == 0 {
				b.WriteString(ch)
			} else {
				b.WriteString(fill.Render(ch))
			}
		}
		lines[r] = b.String()
	}
	return strings.Join(lines, "\n")
}
