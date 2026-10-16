package chart

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/braille"
	"github.com/ows4444/tui/theme"
)

// Canvas is a grid of braille dots to draw on: each terminal cell holds 2
// columns by 4 rows of dots, so a Canvas of w x h cells is 2w x 4h dots, with
// (0, 0) at the top left. Draw points, lines, rectangles and circles, each in
// a colour, then take the rows with Rows or String. Plot and LineChart draw on
// one; it is public for charts and diagrams of your own.
//
// A cell takes the colour of the last shape that set a dot in it, so where two
// shapes meet in one cell the later one shows. The zero Canvas is empty and
// zero-sized; use NewCanvas.
type Canvas struct {
	w, h    int
	pattern []rune
	color   []ansi.Color
}

// NewCanvas returns a Canvas of w x h terminal cells, so 2w x 4h dots. A size
// that is not positive gives an empty Canvas that ignores drawing.
func NewCanvas(w, h int) *Canvas {
	w, h = max(w, 0), max(h, 0)
	return &Canvas{w: w, h: h, pattern: make([]rune, w*h), color: make([]ansi.Color, w*h)}
}

// Size returns the width and height in terminal cells.
func (c *Canvas) Size() (w, h int) { return c.w, c.h }

// DotSize returns the width and height in dots: twice the cells across and four
// times the cells down.
func (c *Canvas) DotSize() (w, h int) { return c.w * 2, c.h * 4 }

// Set lights the dot at column x, row y in colour col (nil takes the theme's
// primary colour when drawn). A dot outside the Canvas is ignored.
func (c *Canvas) Set(x, y int, col ansi.Color) {
	if x < 0 || y < 0 || x >= c.w*2 || y >= c.h*4 {
		return
	}
	i := (y/4)*c.w + x/2
	c.pattern[i] |= braille.Bit[x%2][y%4]
	c.color[i] = col
}

// Line draws a straight line of dots from (x0, y0) to (x1, y1), both ends
// included, stepping along the longer axis so it has no gaps.
func (c *Canvas) Line(x0, y0, x1, y1 int, col ansi.Color) {
	dx, dy := x1-x0, y1-y0
	steps := max(abs(dx), abs(dy))
	if steps == 0 {
		c.Set(x0, y0, col)
		return
	}
	for s := 0; s <= steps; s++ {
		t := float64(s) / float64(steps)
		c.Set(x0+int(float64(dx)*t+sign(float64(dx))*0.5), y0+int(float64(dy)*t+sign(float64(dy))*0.5), col)
	}
}

// Rect draws the outline of the rectangle with its top-left corner at (x, y),
// w dots wide and h dots tall. A size under 1 draws nothing.
func (c *Canvas) Rect(x, y, w, h int, col ansi.Color) {
	if w < 1 || h < 1 {
		return
	}
	c.Line(x, y, x+w-1, y, col)
	c.Line(x, y+h-1, x+w-1, y+h-1, col)
	c.Line(x, y, x, y+h-1, col)
	c.Line(x+w-1, y, x+w-1, y+h-1, col)
}

// Circle draws the outline of the circle of radius r dots around (cx, cy).
// Dots are twice as close across as down in a terminal cell's shape, but a
// dot is square, so the circle looks round as long as a cell is about twice as
// tall as it is wide, which is the usual case.
func (c *Canvas) Circle(cx, cy, r int, col ansi.Color) {
	if r < 0 {
		return
	}
	x, y, d := r, 0, 1-r
	for x >= y {
		for _, p := range [8][2]int{{x, y}, {y, x}, {-y, x}, {-x, y}, {-x, -y}, {-y, -x}, {y, -x}, {x, -y}} {
			c.Set(cx+p[0], cy+p[1], col)
		}
		y++
		if d < 0 {
			d += 2*y + 1
		} else {
			x--
			d += 2*(y-x) + 1
		}
	}
}

// Clear unlights every dot.
func (c *Canvas) Clear() {
	for i := range c.pattern {
		c.pattern[i], c.color[i] = 0, nil
	}
}

// Rows returns the Canvas as one string per row of cells, each w cells wide:
// a braille cell where a dot is lit (a shade of the glyph set under an ASCII
// theme), a space elsewhere, coloured per cell.
func (c *Canvas) Rows(t theme.Theme) []string {
	g := t.GlyphSet()
	rows := make([]string, c.h)
	for r := 0; r < c.h; r++ {
		var b strings.Builder
		for col := 0; col < c.w; col++ {
			i := r*c.w + col
			if c.pattern[i] == 0 {
				b.WriteByte(' ')
				continue
			}
			fg := c.color[i]
			if fg == nil {
				fg = t.Primary
			}
			b.WriteString(ansi.NewStyle().Foreground(fg).Render(braille.DotCell(c.pattern[i], g)))
		}
		rows[r] = b.String()
	}
	return rows
}

// String is Rows joined by newlines.
func (c *Canvas) String(t theme.Theme) string { return strings.Join(c.Rows(t), "\n") }
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
