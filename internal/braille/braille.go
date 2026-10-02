// Package braille is the dot-cell drawing shared by the arc and line charts
// (widgets/chart) and the progress circle (widgets): the 2x4 braille dot
// encoding, its ASCII shade fallback, and the annulus fill behind Gauge and
// ProgressCircle. It is pure string building with no state.
package braille

import (
	"math"
	"math/bits"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// ArcMinWidth is the narrowest width that still fits a semicircle and its
// label; narrower meters fall back to a single-line bar.
const ArcMinWidth = 8

// Bit maps a dot's (column, row) within a 2x4 braille cell to its
// bit in the Unicode braille pattern block (U+2800).
var Bit = [2][4]rune{
	{0x01, 0x02, 0x04, 0x40},
	{0x08, 0x10, 0x20, 0x80},
}

// DotCell draws one 2x4 dot cell. Normally that is the braille character for
// pattern; under an ASCII glyph set it is a shade from g.Shades chosen by how
// many of the eight dots are set, so the chart keeps its shape in plain ASCII.
func DotCell(pattern rune, g theme.Glyphs) string {
	if !g.ASCII() {
		return string(0x2800 + pattern)
	}
	shades := []rune(g.Shades)
	n := bits.OnesCount32(uint32(pattern)) // #nosec G115 -- pattern is a braille dot mask, 0 to 255
	if n == 0 {
		return " "
	}
	return string(shades[min((n-1)*len(shades)/8, len(shades)-1)])
}

// Arc fills a rows x width grid of braille dot cells around a centre
// at dot-coordinates (cx, cy), keeping only dots whose distance from the
// centre falls in the annulus (inner, outer] (in dot units). Each kept dot
// is then classified filled or track by angleFilled, which receives the
// dot's offset from the centre (px, py) and its distance d; a cell is
// rendered in fillStyle if at least half its kept dots are filled, in
// trackStyle otherwise, and left as a space if it has no kept dots.
//
// This is the shared per-dot angle/radius test loop behind both Gauge
// (upper half-plane only, centre on the bottom edge) and ProgressCircle
// (full plane, centre in the middle): only the centre position and the
// angle predicate differ between the two shapes.
func Arc(width, rows int, cx, cy, outer, inner float64, angleFilled func(px, py, d float64) bool, fillStyle, trackStyle ansi.Style, g theme.Glyphs) [][]string {
	cells := make([][]string, rows)
	for r := range cells {
		cells[r] = make([]string, width)
		for c := 0; c < width; c++ {
			var pattern rune
			var filled, total int
			for dx := 0; dx < 2; dx++ {
				for dy := 0; dy < 4; dy++ {
					px := float64(c*2+dx) + 0.5 - cx
					py := cy - (float64(r*4+dy) + 0.5)
					d := math.Hypot(px, py)
					if d > outer || d <= inner {
						continue
					}
					pattern |= Bit[dx][dy]
					total++
					if angleFilled(px, py, d) {
						filled++
					}
				}
			}
			switch {
			case total == 0:
				cells[r][c] = " "
			case filled*2 >= total:
				cells[r][c] = fillStyle.Render(DotCell(pattern, g))
			default:
				cells[r][c] = trackStyle.Render(DotCell(pattern, g))
			}
		}
	}
	return cells
}
