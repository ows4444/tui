package layout

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// borderRing paints the cells of a box's border along the loop around it: the
// top row left to right, the right side down, the bottom row right to left and
// the left side up. The colour at a cell is its place on the loop mapped to a
// blend from one colour to another and back, so the loop has no seam.
type borderRing struct {
	from, to   ansi.RGB
	topW, botW int // cells in the top and bottom rows (0 when that side is off)
	side       int // rows between the top and bottom rows
}

func (r borderRing) on() bool { return r.topW+r.botW+r.side > 0 }

// total is the number of cells on the loop.
func (r borderRing) total() int { return r.topW + r.botW + 2*r.side }

// colourAt is the colour of the n-th cell of the loop.
func (r borderRing) colourAt(n int) ansi.RGB {
	total := r.total()
	if total <= 1 {
		return r.from
	}
	p := float64(n) / float64(total) // 0 at the start, never quite 1: the loop closes on the start
	t := 1 - abs(2*p-1)              // 0 at the start, 1 half way round, back to 0
	return r.from.Mix(r.to, t)
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// paintCells renders s with each cell (a grapheme cluster's runes taken in
// turn) coloured by colour(i), i being the cell's column in s.
func paintCells(s string, colour func(col int) ansi.RGB) string {
	var b strings.Builder
	col := 0
	for _, r := range s {
		cell := string(r)
		w := ansi.Width(cell)
		if w == 0 { // a combining mark rides on the cell before it
			b.WriteString(cell)
			continue
		}
		b.WriteString(ansi.NewStyle().Foreground(colour(col)).Render(cell))
		col += w
	}
	return b.String()
}

func (r borderRing) paintTop(s string) string {
	return paintCells(s, func(col int) ansi.RGB { return r.colourAt(col) })
}

func (r borderRing) paintBottom(s string) string {
	return paintCells(s, func(col int) ansi.RGB {
		// The bottom row runs right to left on the loop.
		return r.colourAt(r.topW + r.side + (r.botW - 1 - col))
	})
}

// paintSide paints one side cell of body row i (0 is the first row under the top).
func (r borderRing) paintSide(s string, i int, rightSide bool) string {
	n := r.topW + i // down the right side
	if !rightSide {
		n = r.topW + r.side + r.botW + (r.side - 1 - i) // up the left side
	}
	return paintCells(s, func(int) ansi.RGB { return r.colourAt(n) })
}
