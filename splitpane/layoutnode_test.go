package splitpane

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

type surf struct {
	g    [][]string
	clip layout.Rect
}

func newSurf(w, h int) *surf {
	s := &surf{clip: layout.Rect{W: w, H: h}}
	for i := 0; i < h; i++ {
		s.g = append(s.g, make([]string, w))
	}
	return s
}
func (s *surf) Clip() layout.Rect     { return s.clip }
func (s *surf) SetClip(r layout.Rect) { s.clip = r }
func (s *surf) Put(x, y int, str string) int {
	n := 0
	for _, r := range ansi.StripANSI(str) {
		if s.clip.Contains(x+n, y) {
			s.g[y][x+n] = string(r)
		}
		n++
	}
	return n
}
func (s *surf) Repeat(x, y, n int, c string) {
	for i := 0; i < n; i++ {
		s.Put(x+i, y, c)
	}
}

func TestMeasure(t *testing.T) {
	m := New(layout.Block("ab\nab"), layout.Block("cdef"))
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 7, H: 2}) {
		t.Fatalf("columns %v", got)
	}
	m.Direction = Rows
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 4, H: 4}) {
		t.Fatalf("rows %v", got)
	}
}

func TestRenderExactSizeAndContent(t *testing.T) {
	m := New(layout.Block("left"), layout.Block("right"))
	m.Min1, m.Min2 = 2, 2
	m.SetTotal(13)
	m.SetPos(5)
	for _, d := range []Direction{Columns, Rows} {
		m.Direction = d
		for _, s := range []layout.Size{{W: 13, H: 3}, {W: 4, H: 2}, {W: 1, H: 1}, {W: 20, H: 1}} {
			lines := strings.Split(m.LayoutNode().Render(s), "\n")
			if len(lines) != s.H {
				t.Fatalf("dir %d %v: %d rows", d, s, len(lines))
			}
			for _, l := range lines {
				if ansi.Width(l) != s.W {
					t.Errorf("dir %d %v: row width %d (%q)", d, s, ansi.Width(l), l)
				}
			}
		}
	}
	m.Direction = Columns
	got := ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 13, H: 2}))
	if got != "left │right  \n     │       " {
		t.Fatalf("render %q", got)
	}
	m.Direction = Rows
	got = ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 5, H: 5}))
	if got != "left \n     \n─────\nright\n     " {
		t.Fatalf("rows render %q", got)
	}
}

func TestDrawCellsMatchesRender(t *testing.T) {
	for _, d := range []Direction{Columns, Rows} {
		m := New(layout.Block("left\nmore"), layout.Block("right"))
		m.Direction = d
		m.Min1, m.Min2 = 2, 2
		sz := layout.Size{W: 12, H: 6}
		m.SetTotal(12)
		m.SetPos(4)
		want := strings.Split(ansi.StripANSI(m.LayoutNode().Render(sz)), "\n")
		s := newSurf(20, 10)
		cn := m.LayoutNode().(layout.CellNode)
		cn.DrawCells(s, layout.Rect{X: 3, Y: 2, W: sz.W, H: sz.H})
		if s.clip != (layout.Rect{W: 20, H: 10}) {
			t.Fatal("clip not restored")
		}
		for y := 0; y < sz.H; y++ {
			var b strings.Builder
			for x := 0; x < sz.W; x++ {
				c := s.g[2+y][3+x]
				if c == "" {
					c = " " // blank cells are expected to be cleared
				}
				b.WriteString(c)
			}
			if b.String() != want[y] {
				t.Errorf("dir %d row %d: %q want %q", d, y, b.String(), want[y])
			}
		}
		if s.g[1][3] != "" || s.g[2][2] != "" || s.g[2][3+12] != "" {
			t.Error("drew outside the rect")
		}
	}
}
