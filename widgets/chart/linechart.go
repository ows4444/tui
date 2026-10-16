package chart

import (
	"github.com/ows4444/tui/theme"
)

// LineChart renders values as a line plotted on a width x height grid of
// character cells, using braille dots (2 sub-columns x 4 sub-rows per cell)
// for roughly double the horizontal and quadruple the vertical resolution
// of a plain block-character plot.
//
// Each value maps to a point: its index spaces it evenly across the
// width*2 dot-columns (first value at column 0, last at the rightmost
// column), and its value is scaled between the slice's own min and max
// into the height*4 dot-rows, min at the bottom and max at the top — the
// same min/max scaling convention as Sparkline, including the flat
// case: when min == max the line renders straight across the vertical
// middle rather than collapsing to one edge.
//
// Consecutive points are connected with a linearly interpolated line
// (stepped along whichever axis has more dots between the two points) so
// the series reads as a continuous line rather than isolated dots. A
// single value renders as one dot; zero values renders "".
//
// Every returned line is at most width columns wide (measured with
// ansi.Width) and there are at most height lines, matching the requested
// grid regardless of how many values are plotted. The line is styled with
// t.Primary.
func LineChart(values []float64, width, height int, t theme.Theme) string {
	if width <= 0 || height <= 0 || len(values) == 0 {
		return ""
	}

	dotCols := width * 2
	dotRows := height * 4

	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	span := max - min

	// Compute each value's dot coordinates.
	px := make([]int, len(values))
	py := make([]int, len(values))
	for i, v := range values {
		if len(values) == 1 {
			px[i] = 0
		} else {
			px[i] = i * (dotCols - 1) / (len(values) - 1)
		}
		if span == 0 {
			py[i] = dotRows / 2
		} else {
			frac := (v - min) / span
			row := int((1-frac)*float64(dotRows-1) + 0.5) // 0 at top, round to nearest
			if row < 0 {
				row = 0
			}
			if row >= dotRows {
				row = dotRows - 1
			}
			py[i] = row
		}
	}

	// Draw the series on a canvas: one dot for a single value, otherwise a
	// straight line between each pair of consecutive points, so the series
	// reads as a continuous line.
	c := NewCanvas(width, height)
	if len(values) == 1 {
		c.Set(px[0], py[0], t.Primary)
	}
	for i := 1; i < len(values); i++ {
		c.Line(px[i-1], py[i-1], px[i], py[i], t.Primary)
	}
	return c.String(t)
}
