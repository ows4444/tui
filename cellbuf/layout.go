package cellbuf

import (
	"strings"
	"unicode/utf8"

	"github.com/ows4444/tui/layout"
)

// Surface adapts a Buffer to layout.CellSurface, so a layout.CellNode tree can
// draw straight into the Buffer's cells (see layout.DrawTo). Coordinates are
// those of the Buffer it was made from. A Surface is not safe for concurrent
// use.
type Surface struct {
	root *Buffer
	view Buffer      // root clipped to clip
	clip layout.Rect // in root coordinates
}

var _ layout.CellSurface = (*Surface)(nil)

// Layout returns a layout.CellSurface that draws into b, clipped to b.
func Layout(b *Buffer) *Surface {
	s := &Surface{root: b}
	s.SetClip(layout.Rect{W: b.r.W, H: b.r.H})
	return s
}

// Clip returns the rectangle drawing is currently clipped to.
func (s *Surface) Clip() layout.Rect { return s.clip }

// SetClip clips later drawing to r, itself clipped to the Buffer.
func (s *Surface) SetClip(r layout.Rect) {
	x0, y0 := max(r.X, 0), max(r.Y, 0)
	x1, y1 := min(r.X+r.W, s.root.r.W), min(r.Y+r.H, s.root.r.H)
	if x1 <= x0 || y1 <= y0 {
		s.clip = layout.Rect{}
		s.view = Buffer{s: s.root.s}
		return
	}
	s.clip = layout.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
	s.view = Buffer{s: s.root.s, r: Rect{X: s.root.r.X + x0, Y: s.root.r.Y + y0, W: x1 - x0, H: y1 - y0}}
}

// Put writes the styled string str (SGR sequences and OSC 8 links allowed, no
// newline) on row y from column x and returns the columns written. A string
// the grid cannot represent is written as plain text with its escapes removed.
func (s *Surface) Put(x, y int, str string) int {
	x, y = x-s.clip.X, y-s.clip.Y
	if str == "" {
		return 0
	}
	if one, ok := s.single(str); ok {
		var a [1]item
		a[0] = one
		return s.view.place(x, y, a[:])
	}
	if isPrintableASCII(str) {
		var a [48]item
		n, wrote := 0, 0
		for i := 0; i < len(str); i++ {
			a[n] = item{text: str[i : i+1], w: 1}
			if n++; n == len(a) || i == len(str)-1 {
				wrote += s.view.place(x+wrote, y, a[:n])
				n = 0
			}
		}
		return wrote
	}
	if strings.IndexByte(str, 0x1b) >= 0 {
		if n, err := s.view.SetStyled(x, y, str); err == nil {
			return n
		}
	}
	return s.view.SetString(x, y, str, 0)
}

// Repeat writes n copies of cluster side by side on row y from column x.
func (s *Surface) Repeat(x, y, n int, cluster string) {
	if n <= 0 {
		return
	}
	x, y = x-s.clip.X, y-s.clip.Y
	if y < 0 || y >= s.view.r.H {
		return
	}
	var one item
	switch it, ok := s.single(cluster); {
	case ok:
		one = it
	case isPrintableASCII(cluster) && len(cluster) == 1:
		one = item{text: cluster, w: 1}
	default:
		var its []item
		if strings.IndexByte(cluster, 0x1b) >= 0 {
			its, _ = s.view.styledItems(cluster)
		}
		if its == nil {
			its = s.view.items(cluster, 0)
		}
		if len(its) == 0 {
			its = []item{{text: " ", w: 1}}
		}
		one = its[0]
	}
	if one.w <= 0 {
		return
	}
	// Skip what lies left of the view and stop at its right edge.
	i := 0
	if x < 0 {
		i = (-x) / one.w
	}
	var a [1]item
	a[0] = one
	for ; x+i*one.w < s.view.r.W && i < n; i++ {
		s.view.place(x+i*one.w, y, a[:])
	}
}

// single reports the item for str when it is exactly one printable
// single-rune cluster outside ASCII (a box-drawing glyph, say), without
// allocating.
func (s *Surface) single(str string) (item, bool) {
	r, size := utf8.DecodeRuneInString(str)
	if size != len(str) || r < 0xa0 || r == utf8.RuneError {
		return item{}, false
	}
	if w := s.view.s.m.Width(str); w == 1 || w == 2 {
		return item{text: str, w: w}, true
	}
	return item{}, false
}
