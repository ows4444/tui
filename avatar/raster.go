package avatar

import (
	"math"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// A pixel is one of four layers, lowest first. A terminal cell holds a two
// by two block of pixels and can show two colours, so a cell is drawn from at
// most two layers.
type layer uint8

const (
	layerNone layer = iota
	layerPlate
	layerBody
	layerEye
	layers
)

// region is a filled outline with its bounding box.
type region struct {
	pts                    []point
	minX, minY, maxX, maxY float64
}

func newRegion(p path) region {
	r := region{pts: p.flatten(), minX: math.Inf(1), minY: math.Inf(1), maxX: math.Inf(-1), maxY: math.Inf(-1)}
	for _, q := range r.pts {
		r.minX, r.maxX = math.Min(r.minX, q.x), math.Max(r.maxX, q.x)
		r.minY, r.maxY = math.Min(r.minY, q.y), math.Max(r.maxY, q.y)
	}
	return r
}

// contains is the even-odd test; every outline here is a simple polygon.
func (r region) contains(x, y float64) bool {
	if x < r.minX || x > r.maxX || y < r.minY || y > r.maxY {
		return false
	}
	in := false
	for i, j := 0, len(r.pts)-1; i < len(r.pts); j, i = i, i+1 {
		a, b := r.pts[i], r.pts[j]
		if (a.y > y) != (b.y > y) && x < (b.x-a.x)*(y-a.y)/(b.y-a.y)+a.x {
			in = !in
		}
	}
	return in
}

// scene is a figure ready to be sampled.
type scene struct {
	plate  *region
	body   []region
	petals []circle
	eyes   [2]region
	eyeAt  [2]point
	// zoom draws the body this many times its size at rest; 0 is 1. It is
	// how the idle loop breathes, and has no effect over a plate.
	zoom float64
	// lift draws the body and the eyes this far down the square; a pose
	// sets it.
	lift float64
}

func (s scene) inBody(x, y float64) bool {
	for _, c := range s.petals {
		if dx, dy := x-c.cx, y-c.cy; dx*dx+dy*dy <= c.r*c.r {
			return true
		}
	}
	for _, b := range s.body {
		if b.contains(x, y) {
			return true
		}
	}
	return false
}

// at is the layer at a point. An eye is part of the body: where a look
// carries it past the outline, it is not drawn.
func (s scene) at(x, y float64) layer {
	// The figure is drawn lift lower: look it up where it is at rest.
	if fy := y - s.lift; s.inBody(x, fy) {
		for _, e := range s.eyes {
			if e.contains(x, fy) {
				return layerEye
			}
		}
		return layerBody
	}
	if s.plate != nil && s.plate.contains(x, y) {
		return layerPlate
	}
	return layerNone
}

// grid is the avatar as pixels: two columns and two rows of them per cell.
type grid struct {
	w, h int // in cells
	px   []layer
}

// samples is the number of sample points along each side of a pixel.
const samples = 3

// bounds returns the square part of the frame that is drawn: the whole frame
// when there is a plate, and otherwise the smallest centred square that holds
// the body, with a little air around it. The frame leaves a wide margin around
// the body, which a few cells cannot afford.
func (s scene) bounds() (cx, cy, side float64) {
	if s.plate != nil {
		return 50, 50, 100
	}
	minX, minY, maxX, maxY := s.extent()
	side = math.Max(maxX-minX, maxY-minY) * bodyAir
	if s.zoom > 0 {
		side /= s.zoom
	}
	return (minX + maxX) / 2, (minY + maxY) / 2, side
}

// extent returns the bounding box of the body at rest.
func (s scene) extent() (minX, minY, maxX, maxY float64) {
	minX, minY, maxX, maxY = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, b := range s.body {
		minX, maxX = math.Min(minX, b.minX), math.Max(maxX, b.maxX)
		minY, maxY = math.Min(minY, b.minY), math.Max(maxY, b.maxY)
	}
	for _, c := range s.petals {
		minX, maxX = math.Min(minX, c.cx-c.r), math.Max(maxX, c.cx+c.r)
		minY, maxY = math.Min(minY, c.cy-c.r), math.Max(maxY, c.cy+c.r)
	}
	return minX, minY, maxX, maxY
}

// room returns how far the body can be lifted (a negative number) and sunk
// before it reaches the edge of the drawn square.
func (s scene) room() (up, down float64) {
	_, cy, side := s.bounds()
	_, minY, _, maxY := s.extent()
	return math.Min(0, cy-side/2-minY), math.Max(0, cy+side/2-maxY)
}

// setLift moves the body and its eyes lift frame units down the drawn square
// (up when negative), stopping at its edge. step, when positive, is the
// height of a pixel: the move is cut down to whole pixels, so a shift too
// small to show does not smear the outline instead.
func (s *scene) setLift(lift, step float64) {
	up, down := s.room()
	lift = math.Max(up, math.Min(down, lift))
	if step > 0 {
		lift = math.Trunc(lift/step) * step
	}
	s.lift = lift
}

// bodyAir is the side of the drawn square as a multiple of the body's larger
// extent.
const bodyAir = 1.04

// newScene prepares f's body, and the plate if there is one, for sampling.
// The eyes are set afterwards, by setEyes, once their pose is known.
func newScene(f figure, plate path) scene {
	s := scene{petals: f.petals}
	if plate != nil {
		r := newRegion(plate)
		s.plate = &r
	}
	for _, e := range f.extra {
		s.body = append(s.body, newRegion(e))
	}
	s.body = append(s.body, newRegion(f.core))
	return s
}

func (s *scene) setEyes(f figure) {
	s.eyeAt = f.eyeAt
	for i, e := range f.eyes {
		s.eyes[i] = newRegion(e)
	}
}

// pixelWidth is the width, in frame units, of one pixel of a w by h cell
// grid. Physical units are cell widths: a pixel is half a unit wide and one
// tall.
func (s scene) pixelWidth(w, h int) float64 {
	_, _, side := s.bounds()
	return side / math.Min(float64(w), float64(2*h)) / 2
}

// Eyes narrower than legibleEye pixels are enlarged to that width, up to
// maxBoost times: in a few cells a true-size eye is less than a pixel, and
// every expression would collapse into the same dot.
const (
	legibleEye = 3.0
	maxBoost   = 2.4
)

// eyeBoost is how much to enlarge f's eyes so they stay legible in a w by h
// cell grid; 1 when they already are.
func (s scene) eyeBoost(f figure, w, h int) float64 {
	width := f.eye[0].rx + f.eye[1].rx // the mean of the two eyes' widths
	return math.Max(1, math.Min(maxBoost, legibleEye*s.pixelWidth(w, h)/width))
}

// rasterize samples the scene into a w by h cell grid. The drawn square (see
// bounds) is fitted, centred, into the cell area, taking a cell as twice as
// tall as it is wide. With keepEyes, each eye takes the pixel under its centre even
// when it is too small to win one.
func (s scene) rasterize(w, h int, keepEyes bool) grid {
	pw, ph := float64(w), float64(2*h)
	bx, by, bside := s.bounds()
	unit := bside / math.Min(pw, ph)
	// (ox, oy) is where the frame's origin falls, in physical units.
	ox, oy := pw/2-bx/unit, ph/2-by/unit

	g := grid{w: w, h: h, px: make([]layer, 4*w*h)}
	for j := 0; j < 2*h; j++ {
		for i := 0; i < 2*w; i++ {
			var n [layers]int
			for b := 0; b < samples; b++ {
				for a := 0; a < samples; a++ {
					x := (float64(i) + (float64(a)+0.5)/samples) * 0.5
					y := float64(j) + (float64(b)+0.5)/samples
					n[s.at((x-ox)*unit, (y-oy)*unit)]++
				}
			}
			g.px[j*2*w+i] = most(n[:])
		}
	}
	if !keepEyes {
		return g
	}
	// An eye narrower than a pixel would lose every vote; give each its
	// pixel, unless a look has carried its centre off the body.
	for _, c := range s.eyeAt {
		i, j := int(math.Floor((c.x/unit+ox)*2)), int(math.Floor((c.y+s.lift)/unit+oy))
		if i >= 0 && j >= 0 && i < 2*w && j < 2*h && s.inBody(c.x, c.y) {
			g.px[j*2*w+i] = layerEye
		}
	}
	return g
}

// most returns the layer with the highest count, the upper one on a tie.
func most(n []int) layer {
	best := layerNone
	for l := range n {
		if n[l] > 0 && n[l] >= n[best] {
			best = layer(l) // #nosec G115 -- l < layers
		}
	}
	return best
}

// cell is one terminal cell: which of its four pixels are ink (bit 0 top
// left, 1 top right, 2 bottom left, 3 bottom right) and the two layers.
type cell struct {
	mask       uint8
	ink, paper layer
}

// quadrant maps an ink mask to its block character. The code points are
// spelled as numbers so that no glyph is a literal: under an ASCII glyph set
// none of them is drawn.
var quadrant = [16]rune{
	' ', 0x2598, 0x259D, 0x2580, 0x2596, 0x258C, 0x259E, 0x259B,
	0x2597, 0x259A, 0x2590, 0x259C, 0x2584, 0x2599, 0x259F, 0x2588,
}

// cellAt reduces the four pixels of cell (cx, cy) to two layers. With three
// or four layers present, the two covering the most pixels are kept and the
// rest join the larger of them. The ink is the body when present, else the
// plate, else the eyes, so that with colour stripped the body is what shows.
func (g grid) cellAt(cx, cy int) cell {
	var px [4]layer
	var n [layers]int
	for k := range px {
		px[k] = g.px[(2*cy+k/2)*2*g.w+2*cx+k%2]
		n[px[k]]++
	}
	a := most(n[:])
	n[a] = 0
	b, two := most(n[:]), false
	for l := range n {
		two = two || n[l] > 0
	}
	if !two {
		b = a
	}
	for k := range px {
		if px[k] != a && px[k] != b {
			px[k] = a
		}
	}

	c := cell{ink: a, paper: b}
	if inkRank(b) > inkRank(a) {
		c.ink, c.paper = b, a
	}
	if a == b && (a == layerNone || a == layerEye) {
		// A cell that is all eye is drawn as background, so it is a hole in
		// the body when colour is stripped.
		c.ink = layerNone
		return c
	}
	for k := range px {
		if px[k] == c.ink {
			c.mask |= 1 << k
		}
	}
	return c
}

func inkRank(l layer) int {
	switch l {
	case layerBody:
		return 3
	case layerPlate:
		return 2
	case layerEye:
		return 1
	}
	return 0
}

// render draws the grid, one styled run per stretch of cells that share
// their colours.
func (g grid) render(p palette, glyphs theme.Glyphs) string {
	colour := [layers]ansi.Color{layerPlate: p.bg, layerBody: p.head, layerEye: p.eye}
	ramp := []rune(glyphs.Shades)
	ascii := glyphs.ASCII() && len(ramp) > 0

	var out strings.Builder
	for cy := 0; cy < g.h; cy++ {
		if cy > 0 {
			out.WriteByte('\n')
		}
		var run strings.Builder
		var ink, paper layer
		flush := func() {
			if run.Len() == 0 {
				return
			}
			st := ansi.NewStyle()
			if ink != layerNone {
				st = st.Foreground(colour[ink])
			}
			if paper != layerNone {
				st = st.Background(colour[paper])
			}
			if ink == layerNone && paper == layerNone {
				out.WriteString(run.String())
			} else {
				out.WriteString(st.Render(run.String()))
			}
			run.Reset()
		}
		for cx := 0; cx < g.w; cx++ {
			c := g.cellAt(cx, cy)
			if c.mask == 0 {
				c.ink = layerNone // nothing is drawn in the foreground colour
			}
			if c.ink != ink || c.paper != paper {
				flush()
				ink, paper = c.ink, c.paper
			}
			switch n := popcount(c.mask); {
			case n == 0:
				run.WriteByte(' ')
			case ascii:
				run.WriteRune(ramp[min((n-1)*len(ramp)/4, len(ramp)-1)])
			default:
				run.WriteRune(quadrant[c.mask])
			}
		}
		flush()
	}
	return out.String()
}

func popcount(m uint8) int {
	n := 0
	for ; m != 0; m &= m - 1 {
		n++
	}
	return n
}
