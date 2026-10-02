package faces

import "math"

// Size picks how big the dot-faces are drawn.
type Size int

const (
	// Small heads are 6x3 cells (12x12 dots).
	Small Size = iota
	// Large heads are 10x5 cells (20x20 dots).
	Large
)

// geom holds every measurement that depends on the size. All of it is in
// dots; the canvas is head plus a margin for ears, antennae and effects.
type geom struct {
	head           int // head is head x head dots
	w, h           int // canvas, in dots (w is a multiple of 2, h of 4)
	ox, oy         int // head's top-left on the canvas
	eyeTop, eyeMax int // top row of the eye band and the tallest eye
	eyeIn          int // left eye's left edge is eyeIn - width/2
	mouthBottom    int // mouths sit on this row and grow upward
}

var geoms = [2]geom{
	Small: {head: 12, w: 24, h: 20, ox: 6, oy: 4, eyeTop: 4, eyeMax: 3, eyeIn: 3, mouthBottom: 9},
	Large: {head: 20, w: 36, h: 36, ox: 8, oy: 8, eyeTop: 7, eyeMax: 5, eyeIn: 6, mouthBottom: 16},
}

func (s Size) geom() geom { return geoms[s] }

// Cells returns the sprite's width and height in terminal cells.
func (s Size) Cells() (w, h int) { g := s.geom(); return g.w / 2, g.h / 4 }

// HeadCells returns just the head's width and height in cells.
func (s Size) HeadCells() (w, h int) { g := s.geom(); return g.head / 2, g.head / 4 }

// u scales a distance given in twelfths of the head (so 12 is one head
// width) to dots at this size. Poses and effects are written in these
// units, which keeps one animation script valid at both sizes.
func (g geom) u(n int) int { return int(math.Round(float64(n) * float64(g.head) / 12)) }
