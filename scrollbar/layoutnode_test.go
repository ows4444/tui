package scrollbar

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// surf is a minimal layout.CellSurface that records plain glyphs.
type surf struct {
	w, h int
	g    [][]string
	clip layout.Rect
}

func newSurf(w, h int) *surf {
	s := &surf{w: w, h: h, clip: layout.Rect{W: w, H: h}}
	for i := 0; i < h; i++ {
		s.g = append(s.g, make([]string, w))
	}
	return s
}
func (s *surf) Clip() layout.Rect     { return s.clip }
func (s *surf) SetClip(r layout.Rect) { s.clip = r }
func (s *surf) Put(x, y int, str string) int {
	if !s.clip.Contains(x, y) {
		return 0
	}
	s.g[y][x] = ansi.StripANSI(str)
	return 1
}
func (s *surf) Repeat(x, y, n int, c string) {
	for i := 0; i < n; i++ {
		s.Put(x+i, y, c)
	}
}

func TestLayoutNodeMeasureRender(t *testing.T) {
	m := New(100, 25)
	m.Length = 6
	n := m.LayoutNode()
	if got := n.Measure(layout.Unconstrained()); got != (layout.Size{W: 1, H: 6}) {
		t.Fatalf("Measure %v", got)
	}
	m.Orientation = Horizontal
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 6, H: 1}) {
		t.Fatalf("Measure %v", got)
	}
	for _, s := range []layout.Size{{W: 1, H: 10}, {W: 3, H: 4}, {W: 1, H: 1}} {
		lines := strings.Split(n.Render(s), "\n")
		if len(lines) != s.H {
			t.Fatalf("%v: %d rows", s, len(lines))
		}
		for _, l := range lines {
			if ansi.Width(l) != s.W {
				t.Errorf("%v: row width %d", s, ansi.Width(l))
			}
		}
	}
	if n.Render(layout.Size{}) != "" {
		t.Error("zero size should be empty")
	}
}

func TestDrawCellsMatchesRender(t *testing.T) {
	for _, o := range []Orientation{Vertical, Horizontal} {
		m := New(60, 20)
		m.Orientation = o
		m.SetOffset(17)
		sz := layout.Size{W: 1, H: 9}
		if o == Horizontal {
			sz = layout.Size{W: 9, H: 1}
		}
		want := strings.Split(ansi.StripANSI(m.LayoutNode().Render(sz)), "\n")
		s := newSurf(12, 12)
		cn := m.LayoutNode().(layout.CellNode)
		cn.DrawCells(s, layout.Rect{X: 2, Y: 1, W: sz.W, H: sz.H})
		if s.clip != (layout.Rect{W: 12, H: 12}) {
			t.Fatal("clip not restored")
		}
		for y := 0; y < sz.H; y++ {
			var got strings.Builder
			for x := 0; x < sz.W; x++ {
				got.WriteString(s.g[1+y][2+x])
			}
			if got.String() != want[y] {
				t.Errorf("orientation %d row %d: %q want %q", o, y, got.String(), want[y])
			}
		}
		// nothing outside r
		if s.g[0][2] != "" || s.g[1][1] != "" {
			t.Error("drew outside rect")
		}
	}
}
