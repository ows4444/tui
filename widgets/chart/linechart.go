package chart

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/braille"
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

	// Mark every dot the line passes through, connecting consecutive
	// points with a linear interpolation.
	lit := make(map[[2]int]bool)
	lit[[2]int{px[0], py[0]}] = true
	for i := 1; i < len(values); i++ {
		plotLine(lit, px[i-1], py[i-1], px[i], py[i])
	}

	fill := ansi.NewStyle().Foreground(t.Primary)

	lines := make([]string, height)
	for r := 0; r < height; r++ {
		cols := make([]string, width)
		for c := 0; c < width; c++ {
			var pattern rune
			var any bool
			for dx := 0; dx < 2; dx++ {
				for dy := 0; dy < 4; dy++ {
					col := c*2 + dx
					row := r*4 + dy
					if lit[[2]int{col, row}] {
						pattern |= braille.Bit[dx][dy]
						any = true
					}
				}
			}
			if any {
				cols[c] = fill.Render(braille.DotCell(pattern, t.GlyphSet()))
			} else {
				cols[c] = " "
			}
		}
		lines[r] = strings.Join(cols, "")
	}
	return strings.Join(lines, "\n")
}

// plotLine marks every dot on the straight line between (x0,y0) and
// (x1,y1) as lit, stepping along whichever axis spans more dots so the
// interpolation stays continuous (no gaps) in both directions.
func plotLine(lit map[[2]int]bool, x0, y0, x1, y1 int) {
	dx := x1 - x0
	dy := y1 - y0
	steps := abs(dx)
	if abs(dy) > steps {
		steps = abs(dy)
	}
	if steps == 0 {
		lit[[2]int{x0, y0}] = true
		return
	}
	for s := 0; s <= steps; s++ {
		t := float64(s) / float64(steps)
		x := x0 + int(float64(dx)*t+sign(float64(dx))*0.5)
		y := y0 + int(float64(dy)*t+sign(float64(dy))*0.5)
		lit[[2]int{x, y}] = true
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func sign(f float64) float64 {
	if f < 0 {
		return -1
	}
	if f > 0 {
		return 1
	}
	return 0
}
