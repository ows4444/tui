package hittest_test

import (
	"testing"

	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
)

// Criterion #56: every cell inside a named node's rectangle reports its name
// and no cell outside does.
func TestHitMapReportsNameInsideRectOnly(t *testing.T) {
	ui := layout.Column(1,
		layout.FlexChild{Node: layout.Named("head", layout.Block("title")), Basis: 1},
		layout.FlexChild{Node: layout.Row(2,
			layout.FlexChild{Node: layout.Named("list", layout.Block("a\nb\nc")), Basis: 7},
			layout.FlexChild{Node: layout.Named("body", layout.Block("body")), Grow: 1},
		), Grow: 1},
	)
	s := layout.Size{W: 20, H: 8}
	m := hittest.HitMap(ui, s)
	rects := map[string]layout.Rect{}
	for _, p := range layout.Rects(ui, s) {
		if p.Name != "" {
			rects[p.Name] = p.Rect
		}
	}
	if len(rects) != 3 || m.Len() != 3 {
		t.Fatalf("rects %v, map len %d; want 3 named", rects, m.Len())
	}
	for y := -1; y <= s.H; y++ {
		for x := -1; x <= s.W; x++ {
			h, ok := m.At(x, y)
			inAny := false
			for name, r := range rects {
				if !r.Contains(x, y) {
					continue
				}
				inAny = true
				if !ok || h.ID != name || h.LX != x-r.X || h.LY != y-r.Y {
					t.Fatalf("(%d,%d) in %s %v: got %+v ok=%v", x, y, name, r, h, ok)
				}
			}
			if !inAny && ok {
				t.Fatalf("(%d,%d) outside every named node reported %q", x, y, h.ID)
			}
		}
	}
}

// Criterion #57: overlapping nodes report the topmost.
func TestHitMapOverlapTopmost(t *testing.T) {
	block := func(c string, w, h int) layout.Node {
		row := ""
		for i := 0; i < w; i++ {
			row += c
		}
		rows := make([]string, h)
		for i := range rows {
			rows[i] = row
		}
		return layout.Block(joinRows(rows))
	}
	ui := layout.Stack(
		layout.Layer(9, layout.Absolute(2, 1, layout.Named("dialog", block("d", 4, 3)))),
		layout.Layer(1, layout.Named("page", block("p", 8, 5))),
	)
	s := layout.Size{W: 8, H: 5}
	m := hittest.HitMap(ui, s)
	for _, c := range []struct {
		x, y int
		want string
	}{{0, 0, "page"}, {2, 0, "page"}, {2, 1, "dialog"}, {5, 3, "dialog"}, {6, 1, "page"}, {3, 4, "page"}} {
		h, ok := m.At(c.x, c.y)
		if !ok || h.ID != c.want {
			t.Errorf("At(%d,%d) = %q, %v; want %q", c.x, c.y, h.ID, ok, c.want)
		}
		if name, _, _, _ := layout.NamedAt(ui, s, c.x, c.y); name != c.want {
			t.Errorf("NamedAt(%d,%d) = %q; want %q", c.x, c.y, name, c.want)
		}
	}
	// A nested named node is on top of its named parent.
	nested := layout.Named("outer", layout.Column(0, layout.FlexChild{Node: layout.Named("inner", layout.Block("x")), Basis: 1}))
	if h, _ := hittest.HitMap(nested, layout.Size{W: 3, H: 2}).At(0, 0); h.ID != "inner" {
		t.Errorf("nested At(0,0) = %q, want inner", h.ID)
	}
}

func joinRows(rows []string) string {
	out := ""
	for i, r := range rows {
		if i > 0 {
			out += "\n"
		}
		out += r
	}
	return out
}
