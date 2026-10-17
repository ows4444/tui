// Package fx applies effects to a region of cells: fade, dissolve, slide and
// sweep, for a panel that appears or goes away. An Effect rewrites a
// cellbuf.Buffer that already holds the finished picture, as a function of a
// progress p from 0 (nothing shown) to 1 (the picture as it is). Wrap puts an
// Effect around any layout.Node; Out turns an effect that shows into one that
// hides; Progress turns elapsed time into a p, easing it and jumping to 1 under
// a reduced-motion preference.
//
//	view := fx.Wrap(panel, fx.Fade{}, fx.Progress(elapsed, 300*time.Millisecond, motion.OutCubic, pref))
//
// An effect never changes the buffer's size or the text it holds once p is 1.
//
// Stability: experimental. Its API may change in any minor release.
package fx

import (
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
)

// Effect rewrites the cells of r in dst for progress p. p is clamped to [0, 1]
// by the callers in this package; at 1 an effect leaves dst untouched.
type Effect interface {
	Apply(dst *cellbuf.Buffer, r cellbuf.Rect, p float64)
}

// Direction is the edge an effect starts from.
type Direction int

// The edges. FromLeft is the zero value.
const (
	FromLeft Direction = iota
	FromRight
	FromTop
	FromBottom
)

// Progress turns time into the p an Effect takes: elapsed over d, shaped by
// ease (nil is linear), clamped to [0, 1]. With a Reduced preference, or a d of
// 0 or less, it is 1 at once, so the picture simply appears.
func Progress(elapsed, d time.Duration, ease func(float64) float64, pref motion.Preference) float64 {
	if pref.Reduced() || d <= 0 || elapsed >= d {
		return 1
	}
	if elapsed <= 0 {
		return 0
	}
	t := float64(elapsed) / float64(d)
	if ease != nil {
		t = ease(t)
	}
	return clamp(t)
}

func clamp(p float64) float64 { return max(0, min(p, 1)) }

// Out returns e run backwards: at p of 0 the picture is whole and at 1 it has
// gone, so a panel that closes uses the same effect it opened with.
func Out(e Effect) Effect { return out{e} }

type out struct{ e Effect }

func (o out) Apply(dst *cellbuf.Buffer, r cellbuf.Rect, p float64) {
	o.e.Apply(dst, r, 1-clamp(p))
}

// Default colours for a cell that names none: what a terminal usually draws.
var defaultFG = ansi.RGB{R: 229, G: 229, B: 229}

// region is r clipped to dst's own cells.
func region(dst *cellbuf.Buffer, r cellbuf.Rect) cellbuf.Rect {
	x0, y0 := max(r.X, 0), max(r.Y, 0)
	x1, y1 := min(r.X+r.W, dst.Width()), min(r.Y+r.H, dst.Height())
	if x1 <= x0 || y1 <= y0 {
		return cellbuf.Rect{}
	}
	return cellbuf.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// restyle gives the cell at (x, y) the style id, keeping its text.
func restyle(dst *cellbuf.Buffer, x, y int, c cellbuf.Cell, id cellbuf.StyleID) {
	if c.Width == 0 {
		return // the second half of a wide cluster follows its head
	}
	dst.SetString(x, y, c.Cluster, id)
}

// blankCell clears the cell at (x, y), and the second half of a wide cluster.
func blankCell(dst *cellbuf.Buffer, x, y int, c cellbuf.Cell) {
	dst.SetString(x, y, " ", 0)
	if c.Width == 2 {
		dst.SetString(x+1, y, " ", 0)
	}
}

// blend is c moved toward to by f (0 keeps c, 1 is to).
func blend(c cellbuf.Color, def, to ansi.RGB, f float64) cellbuf.Color {
	rgb, ok := c.RGB()
	if !ok {
		rgb = def
	}
	m := rgb.Mix(to, f)
	return cellbuf.RGB(m.R, m.G, m.B)
}

// faded caches the style a style becomes when blended toward to by f.
type faded struct {
	dst *cellbuf.Buffer
	to  ansi.RGB
	f   float64
	ids map[cellbuf.StyleID]cellbuf.StyleID
}

func newFaded(dst *cellbuf.Buffer, to ansi.RGB, f float64) *faded {
	return &faded{dst: dst, to: to, f: clamp(f), ids: map[cellbuf.StyleID]cellbuf.StyleID{}}
}

func (fd *faded) id(id cellbuf.StyleID) cellbuf.StyleID {
	if n, ok := fd.ids[id]; ok {
		return n
	}
	st := fd.dst.Style(id)
	st.FG = blend(st.FG, defaultFG, fd.to, fd.f)
	st.BG = blend(st.BG, fd.to, fd.to, fd.f)
	n := fd.dst.StyleID(st)
	fd.ids[id] = n
	return n
}

// Fade blends the colours of every cell toward To as p falls: at 0 the text is
// the colour of the background and cannot be told from it. A cell with no
// background of its own is taken to be To already. Attributes and text stay.
type Fade struct {
	// To is the colour to fade to, normally the background behind the region.
	// The zero value is black.
	To ansi.RGB
}

// Apply implements Effect.
func (f Fade) Apply(dst *cellbuf.Buffer, r cellbuf.Rect, p float64) {
	p = clamp(p)
	r = region(dst, r)
	if p >= 1 || r.W == 0 {
		return
	}
	fd := newFaded(dst, f.To, 1-p)
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			c := dst.At(x, y)
			restyle(dst, x, y, c, fd.id(c.Style))
		}
	}
}

// Dissolve hides cells one by one, in an order fixed by Seed, so that a share p
// of them is showing: a cell that is hidden at p stays hidden at any lower p.
type Dissolve struct {
	// Seed picks the order; the same Seed gives the same dissolve.
	Seed uint64
}

// Apply implements Effect.
func (d Dissolve) Apply(dst *cellbuf.Buffer, r cellbuf.Rect, p float64) {
	p = clamp(p)
	r = region(dst, r)
	if p >= 1 || r.W == 0 {
		return
	}
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			c := dst.At(x, y)
			if c.Width == 0 {
				continue
			}
			if rank(x, y, d.Seed) >= p {
				blankCell(dst, x, y, c)
			}
		}
	}
}

// rank is a number in [0, 1) fixed by the cell and the seed.
func rank(x, y int, seed uint64) float64 {
	h := seed ^ uint64(x)*0x9e3779b97f4a7c15 ^ uint64(y)*0xc2b2ae3d27d4eb4f // #nosec G115 -- cell coordinates are small and non-negative
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return float64(h>>11) / (1 << 53)
}

// Slide moves the picture in from an edge: at p of 0 it is wholly outside the
// region on the side named by From, and it moves across until it is in place.
// What is outside the region is cut off.
type Slide struct {
	From Direction
}

// Apply implements Effect.
func (s Slide) Apply(dst *cellbuf.Buffer, r cellbuf.Rect, p float64) {
	p = clamp(p)
	r = region(dst, r)
	if p >= 1 || r.W == 0 {
		return
	}
	type cell struct {
		c    cellbuf.Cell
		x, y int
	}
	saved := make([]cell, 0, r.W*r.H)
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			if c := dst.At(r.X+x, r.Y+y); c.Width > 0 {
				saved = append(saved, cell{c, x, y})
			}
		}
	}
	dx, dy := 0, 0
	switch s.From {
	case FromLeft:
		dx = -int((1-p)*float64(r.W) + 0.5)
	case FromRight:
		dx = int((1-p)*float64(r.W) + 0.5)
	case FromTop:
		dy = -int((1-p)*float64(r.H) + 0.5)
	case FromBottom:
		dy = int((1-p)*float64(r.H) + 0.5)
	}
	view := dst.Sub(r) // writes are cut off at the region's edge
	view.Fill(view.Bounds(), " ", 0)
	for _, s := range saved {
		view.SetString(s.x+dx, s.y+dy, s.c.Cluster, s.c.Style)
	}
}

// Sweep reveals the picture with an edge that crosses the region from the side
// named by From. Beyond the edge nothing shows; the Soft cells behind it fade
// in, so the edge is a gradient and not a cut.
type Sweep struct {
	From Direction
	// Soft is the width of the gradient in cells; 0 is a hard edge.
	Soft int
	// To is the colour the gradient fades to, as in Fade.
	To ansi.RGB
}

// Apply implements Effect.
func (s Sweep) Apply(dst *cellbuf.Buffer, r cellbuf.Rect, p float64) {
	p = clamp(p)
	r = region(dst, r)
	if p >= 1 || r.W == 0 {
		return
	}
	soft := max(s.Soft, 0)
	extent := r.W
	if s.From == FromTop || s.From == FromBottom {
		extent = r.H
	}
	edge := p * float64(extent+soft) // how far the revealed part reaches
	ids := map[int]*faded{}
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			c := dst.At(x, y)
			if c.Width == 0 {
				continue
			}
			var d int // distance from the starting edge
			switch s.From {
			case FromLeft:
				d = x - r.X
			case FromRight:
				d = r.X + r.W - 1 - x
			case FromTop:
				d = y - r.Y
			default:
				d = r.Y + r.H - 1 - y
			}
			behind := edge - float64(d) // cells the edge is past this one
			switch {
			case behind <= 0:
				blankCell(dst, x, y, c)
			case behind >= float64(soft):
			default:
				k := int(behind*8) + 1 // eight steps keep the style table small
				fd, ok := ids[k]
				if !ok {
					fd = newFaded(dst, s.To, 1-float64(k-1)/8/float64(soft))
					ids[k] = fd
				}
				restyle(dst, x, y, c, fd.id(c.Style))
			}
		}
	}
}

// Wrap returns node with e applied at progress p. The node is drawn into a cell
// grid of the size it is given and the effect rewrites that grid, so any
// layout.Node can be faded, slid or swept. At a p of 1 the node is drawn as it
// is.
func Wrap(node layout.Node, e Effect, p float64) layout.Node {
	return wrapped{node: node, e: e, p: clamp(p)}
}

type wrapped struct {
	node layout.Node
	e    Effect
	p    float64
}

func (w wrapped) Measure(c layout.Constraints) layout.Size { return w.node.Measure(c) }

// grid draws the node into a buffer of size s and applies the effect.
func (w wrapped) grid(s layout.Size) *cellbuf.Buffer {
	b := cellbuf.New(s.W, s.H)
	layout.DrawTo(w.node, cellbuf.Layout(b), layout.Tight(s))
	w.e.Apply(b, b.Bounds(), w.p)
	return b
}

func (w wrapped) Render(s layout.Size) string {
	if w.p >= 1 || s.W <= 0 || s.H <= 0 {
		return w.node.Render(s)
	}
	return w.grid(s).String()
}

// DrawCells implements layout.CellNode.
func (w wrapped) DrawCells(dst layout.CellSurface, r layout.Rect) {
	if r.Empty() {
		return
	}
	if cn, ok := w.node.(layout.CellNode); ok && w.p >= 1 {
		cn.DrawCells(dst, r)
		return
	}
	for i, l := range w.grid(layout.Size{W: r.W, H: r.H}).Lines() {
		dst.Put(r.X, r.Y+i, l)
	}
}

var _ layout.CellNode = wrapped{}
