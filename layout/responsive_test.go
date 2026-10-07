package layout

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func TestResponsiveChangesAcrossEachBreak(t *testing.T) {
	r := Responsive([]Break{{MaxW: 10}, {MaxW: 20}}, fill{'a', 1, 1}, fill{'b', 1, 1}, fill{'c', 1, 1})
	for _, tc := range []struct {
		w    int
		want byte
	}{{1, 'a'}, {9, 'a'}, {10, 'a'}, {11, 'b'}, {19, 'b'}, {20, 'b'}, {21, 'c'}, {80, 'c'}} {
		out := r.Render(Size{tc.w, 1})
		if out != strings.Repeat(string(tc.want), tc.w) {
			t.Errorf("Render width %d = %q, want %c", tc.w, out, tc.want)
		}
		if got := r.Measure(Constraints{MaxW: tc.w, MaxH: 5}); got.W != 1 {
			t.Errorf("Measure width %d = %v", tc.w, got)
		}
	}
	// Measure follows the node chosen by MaxW.
	wide := Responsive([]Break{{MaxW: 10}}, Block("ab"), Block("abcdef"))
	if got := wide.Measure(Loose(Size{10, 5})); got != (Size{2, 1}) {
		t.Errorf("narrow Measure = %v", got)
	}
	if got := wide.Measure(Loose(Size{11, 5})); got != (Size{6, 1}) {
		t.Errorf("wide Measure = %v", got)
	}
}

func TestResponsiveEdges(t *testing.T) {
	if got := Responsive(nil).Render(Size{3, 2}); got != "   \n   " {
		t.Errorf("empty = %q", got)
	}
	last := Responsive([]Break{{MaxW: 2}, {MaxW: 4}}, fill{'a', 1, 1}, fill{'b', 1, 1})
	if got := last.Render(Size{6, 1}); got != "bbbbbb" {
		t.Errorf("past every break = %q, want last node", got)
	}
	any := Responsive([]Break{{}}, fill{'a', 1, 1})
	if got := any.Render(Size{5, 1}); got != "aaaaa" {
		t.Errorf("zero MaxW = %q", got)
	}
}

func TestRectsSeeThroughResponsive(t *testing.T) {
	r := Responsive([]Break{{MaxW: 10}}, Named("small", fill{'s', 1, 1}), Named("big", fill{'b', 1, 1}))
	if rc, ok := RectOf(r, Size{8, 2}, "small"); !ok || rc != (Rect{0, 0, 8, 2}) {
		t.Errorf("small = %+v %v", rc, ok)
	}
	if _, ok := RectOf(r, Size{12, 2}, "small"); ok {
		t.Error("small found at wide width")
	}
	if rc, ok := RectOf(r, Size{12, 2}, "big"); !ok || rc != (Rect{0, 0, 12, 2}) {
		t.Errorf("big = %+v %v", rc, ok)
	}
}

// respTree builds random trees with Responsive at every level.
func respTree(g *treeGen, depth int) Node {
	rng := g.rng
	if depth <= 0 {
		return g.leaf()
	}
	switch rng.Intn(4) {
	case 0:
		n := 1 + rng.Intn(3)
		var breaks []Break
		nodes := make([]Node, n+rng.Intn(2))
		w := 0
		for i := 0; i < n; i++ {
			w += 1 + rng.Intn(10)
			breaks = append(breaks, Break{MaxW: w, MaxH: rng.Intn(8)})
		}
		for i := range nodes {
			nodes[i] = respTree(g, depth-1)
		}
		return Responsive(breaks, nodes...)
	case 1:
		kids := []FlexChild{{Node: respTree(g, depth-1), Grow: 1}, {Node: respTree(g, depth-1)}}
		if rng.Intn(2) == 0 {
			return Row(rng.Intn(2), kids...)
		}
		return Column(rng.Intn(2), kids...)
	case 2:
		return Named("n", respTree(g, depth-1))
	}
	return g.node(depth)
}

func TestResponsiveRandomSizeAndRects(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	for trial := 0; trial < 6000; trial++ {
		root := respTree(&treeGen{rng: rng}, 1+rng.Intn(4))
		s := Size{W: 1 + rng.Intn(40), H: 1 + rng.Intn(12)}
		out := root.Render(s)
		rows := strings.Split(out, "\n")
		if len(rows) != s.H {
			t.Fatalf("trial %d: %d rows, want %d\n%q", trial, len(rows), s.H, out)
		}
		for _, l := range rows {
			if ansi.Width(l) != s.W {
				t.Fatalf("trial %d: row %q width %d, want %d", trial, l, ansi.Width(l), s.W)
			}
		}
		cells := cellsOf(out)
		for _, p := range Rects(root, s) {
			f, ok := p.Node.(fill)
			if !ok {
				continue
			}
			count := 0
			for y, row := range cells {
				for x, r := range row {
					if r == f.ch {
						count++
						if !p.Rect.Contains(x, y) {
							t.Fatalf("trial %d: %q at (%d,%d) outside %+v\n%s", trial, f.ch, x, y, p.Rect, out)
						}
					}
				}
			}
			if want := p.Rect.W * p.Rect.H; p.Rect.Empty() && count != 0 || !p.Rect.Empty() && count != want {
				t.Fatalf("trial %d: %q drawn in %d cells, rect %+v covers %d\n%s", trial, f.ch, count, p.Rect, want, out)
			}
		}
		if m := root.Measure(Loose(s)); m.W > s.W || m.H > s.H {
			t.Fatalf("trial %d: Measure %v exceeds %v", trial, m, s)
		}
	}
}

// Responsive shows a compact layout on a narrow terminal and a fuller one on a
// wide one.

func TestResponsiveHeightSkipsAndZeroAdmitsAny(t *testing.T) {
	r := Responsive([]Break{{MaxH: 5}, {MaxW: 10, MaxH: 20}, {}}, fill{'a', 1, 1}, fill{'b', 1, 1}, fill{'c', 1, 1})
	for _, tc := range []struct {
		s    Size
		want rune
	}{
		{Size{4, 5}, 'a'},     // within MaxH
		{Size{4, 6}, 'b'},     // taller than first MaxH: skipped
		{Size{11, 6}, 'c'},    // too wide for second
		{Size{4, 21}, 'c'},    // too tall for second
		{Size{500, 900}, 'c'}, // zero admits any
	} {
		if got := []rune(strings.Split(r.Render(tc.s), "\n")[0])[0]; got != tc.want {
			t.Errorf("%+v: got %q want %q", tc.s, got, tc.want)
		}
	}
	if m := r.Measure(Loose(Size{W: 4, H: 6})); m != (Size{1, 1}) {
		t.Errorf("measure = %+v", m)
	}
	if a, b := (Break{MaxH: 1}), (Break{MaxH: 1}); a != b {
		t.Error("Break must stay comparable")
	}
}
