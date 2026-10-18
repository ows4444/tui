package main

import "strings"

type pt struct{ x, y int }

// grid is the picture: w x h pixels, 0 for empty and 1 to 8 for a palette
// colour. Every method that changes it works on a copy.
type grid struct {
	w, h int
	px   []uint8
}

func newGrid(w, h int) grid { return grid{w, h, make([]uint8, max(0, w)*max(0, h))} }

func (g grid) in(p pt) bool { return p.x >= 0 && p.y >= 0 && p.x < g.w && p.y < g.h }

func (g grid) at(p pt) uint8 { return g.px[p.y*g.w+p.x] }

func (g grid) clone() grid {
	g.px = append([]uint8(nil), g.px...)
	return g
}

// resize returns the picture on a w x h sheet, cut or padded at the right and
// bottom.
func (g grid) resize(w, h int) grid {
	n := newGrid(w, h)
	for y := 0; y < min(h, g.h); y++ {
		copy(n.px[y*w:y*w+min(w, g.w)], g.px[y*g.w:])
	}
	return n
}

func (g grid) empty() bool {
	for _, c := range g.px {
		if c != 0 {
			return false
		}
	}
	return true
}

// stamp sets a size x size square of pixels centred on p. It changes g in
// place; callers clone first.
func (g grid) stamp(p pt, size int, c uint8) {
	lo := -(size - 1) / 2
	for dy := lo; dy < lo+size; dy++ {
		for dx := lo; dx < lo+size; dx++ {
			if q := (pt{p.x + dx, p.y + dy}); g.in(q) {
				g.px[q.y*g.w+q.x] = c
			}
		}
	}
}

// line draws from a to b with Bresenham's algorithm, stamping at every step.
func (g grid) line(a, b pt, size int, c uint8) {
	dx, dy := abs(b.x-a.x), -abs(b.y-a.y)
	sx, sy := sign(b.x-a.x), sign(b.y-a.y)
	e := dx + dy
	for {
		g.stamp(a, size, c)
		if a == b {
			return
		}
		switch e2 := 2 * e; {
		case e2 >= dy && e2 <= dx:
			e += dy + dx
			a.x, a.y = a.x+sx, a.y+sy
		case e2 >= dy:
			e += dy
			a.x += sx
		default:
			e += dx
			a.y += sy
		}
	}
}

// rect draws the outline of the rectangle with corners a and b.
func (g grid) rect(a, b pt, size int, c uint8) {
	g.line(pt{a.x, a.y}, pt{b.x, a.y}, size, c)
	g.line(pt{b.x, a.y}, pt{b.x, b.y}, size, c)
	g.line(pt{b.x, b.y}, pt{a.x, b.y}, size, c)
	g.line(pt{a.x, b.y}, pt{a.x, a.y}, size, c)
}

// fill replaces the connected area of p's colour with c (not across diagonals).
func (g grid) fill(p pt, c uint8) {
	if !g.in(p) || g.at(p) == c {
		return
	}
	old := g.at(p)
	stack := []pt{p}
	for len(stack) > 0 {
		q := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !g.in(q) || g.at(q) != old {
			continue
		}
		g.px[q.y*g.w+q.x] = c
		stack = append(stack, pt{q.x + 1, q.y}, pt{q.x - 1, q.y}, pt{q.x, q.y + 1}, pt{q.x, q.y - 1})
	}
}

// glyphs are the characters a saved text picture uses, one per colour.
const glyphs = " .:-=+*#%"

// text is the picture as plain text: one glyph per pixel, trailing blanks and
// blank lines at the end cut.
func (g grid) text() string {
	var rows []string
	for y := 0; y < g.h; y++ {
		var b strings.Builder
		for x := 0; x < g.w; x++ {
			b.WriteByte(glyphs[g.at(pt{x, y})%uint8(len(glyphs))])
		}
		rows = append(rows, strings.TrimRight(b.String(), " "))
	}
	return strings.TrimRight(strings.Join(rows, "\n"), "\n") + "\n"
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}
