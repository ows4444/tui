package layout

import (
	"math/rand"
	"strings"
	"testing"
)

// fill is a leaf that draws its allotted size in one rune, so the cells it
// occupies in a rendered frame can be found exactly.
type fill struct {
	ch   rune
	w, h int // natural size
}

func (f fill) Measure(c Constraints) Size { return c.Constrain(Size{W: f.w, H: f.h}) }

func (f fill) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	row := strings.Repeat(string(f.ch), s.W)
	rows := make([]string, s.H)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

func TestRectsColumnWithGapsMatchesTheSolver(t *testing.T) {
	top := fill{'a', 5, 1}
	mid := fill{'b', 3, 2}
	bot := fill{'c', 4, 1}
	col := Column(1,
		FlexChild{Node: top},
		FlexChild{Node: mid, Grow: 1},
		FlexChild{Node: bot},
	)
	got := Rects(col, Size{W: 8, H: 9})
	want := []Rect{
		{0, 0, 8, 9},
		{0, 0, 8, 1}, // top
		{0, 2, 8, 4}, // mid takes the 4 spare rows
		{0, 7, 8, 1}, // bot; the gaps are rows 1 and 6
	}
	// 9 = 1 + gap + 4 + gap + 1 + ... check by the parts: 1+1+4+1+1 = 8, one row spare.
	want[2] = Rect{0, 2, 8, 5}
	want[3] = Rect{0, 8, 8, 1}
	if len(got) != len(want) {
		t.Fatalf("%d entries, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Rect != w {
			t.Errorf("entry %d rect = %+v, want %+v", i, got[i].Rect, w)
		}
	}
	if got[0].Depth != 0 || got[1].Depth != 1 {
		t.Errorf("depths = %d, %d", got[0].Depth, got[1].Depth)
	}
}

func TestRectsBoxNodeOffsetsByBorderAndPadding(t *testing.T) {
	box := BoxNode(NewBox().Border(NormalBorder()).Padding(1, 2, 3, 4), fill{'x', 1, 1})
	got := Rects(box, Size{W: 20, H: 10})
	// border 1 + padding left 4 = x 5; border 1 + padding top 1 = y 2;
	// inner width 20-2-4-2 = 12, inner height 10-2-1-3 = 4.
	if len(got) != 2 || got[1].Rect != (Rect{5, 2, 12, 4}) {
		t.Errorf("rects = %+v", got)
	}
}

func TestRectsGridCells(t *testing.T) {
	g := GridNode([]Track{{Size: 3}, {Grow: 1}}, 1,
		fill{'a', 1, 1}, fill{'b', 1, 1},
		fill{'c', 1, 2}, fill{'d', 1, 1},
	)
	got := Rects(g, Size{W: 10, H: 5})
	want := []Rect{{0, 0, 10, 5}, {0, 0, 3, 1}, {4, 0, 6, 1}, {0, 2, 3, 2}, {4, 2, 6, 2}}
	if len(got) != len(want) {
		t.Fatalf("%d entries, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Rect != w {
			t.Errorf("cell %d rect = %+v, want %+v", i, got[i].Rect, w)
		}
	}
}

func TestRectsRowCrossAlignAndJustify(t *testing.T) {
	row := RowJustify(0, JustifyCenter,
		FlexChild{Node: fill{'a', 2, 1}, CrossAlign: CrossStart},
		FlexChild{Node: fill{'b', 2, 1}, CrossAlign: CrossEnd},
	)
	got := Rects(row, Size{W: 10, H: 3})
	// 6 spare columns, centred: 3 before. a at the top, b at the bottom.
	if got[1].Rect != (Rect{3, 0, 2, 1}) || got[2].Rect != (Rect{5, 2, 2, 1}) {
		t.Errorf("rects = %+v", got)
	}
}

func TestNamedAndRectOf(t *testing.T) {
	list := Named("list", fill{'l', 4, 2})
	ui := Row(1,
		FlexChild{Node: list},
		FlexChild{Node: Named("side", fill{'s', 3, 2}), Grow: 1},
	)
	if r, ok := RectOf(ui, Size{W: 12, H: 2}, "list"); !ok || r != (Rect{0, 0, 4, 2}) {
		t.Errorf("RectOf(list) = %+v, %v", r, ok)
	}
	if r, ok := RectOf(ui, Size{W: 12, H: 2}, "side"); !ok || r != (Rect{5, 0, 7, 2}) {
		t.Errorf("RectOf(side) = %+v, %v", r, ok)
	}
	if _, ok := RectOf(ui, Size{W: 12, H: 2}, "missing"); ok {
		t.Error("RectOf(missing) reported a match")
	}
	if _, ok := RectOf(ui, Size{W: 12, H: 2}, ""); ok {
		t.Error("an empty name matched")
	}
	// Named changes nothing about layout.
	plain := Row(1, FlexChild{Node: fill{'l', 4, 2}}, FlexChild{Node: fill{'s', 3, 2}, Grow: 1})
	if a, b := Draw(ui, Unconstrained()), Draw(plain, Unconstrained()); a != b {
		t.Errorf("Named changed the output:\n%q\n%q", a, b)
	}
}

func TestRectsOverflowIsClippedToTheParent(t *testing.T) {
	// The child wants 20 columns but the row is 6 wide.
	row := Row(0, FlexChild{Node: fill{'a', 20, 1}})
	got := Rects(row, Size{W: 6, H: 2})
	if got[1].Rect != (Rect{0, 0, 6, 2}) {
		t.Errorf("overflowing child rect = %+v, want it clipped to 6x2", got[1].Rect)
	}
	// A child pushed wholly outside reports an empty rectangle.
	push := Row(0,
		FlexChild{Node: fill{'a', 6, 1}, Shrink: 0},
		FlexChild{Node: fill{'b', 6, 1}, Shrink: 0},
	)
	got = Rects(push, Size{W: 6, H: 1})
	if !got[2].Rect.Empty() {
		t.Errorf("clipped-away child rect = %+v, want empty", got[2].Rect)
	}
}

func TestRectsOfALeafAndOfNothing(t *testing.T) {
	if got := Rects(fill{'a', 1, 1}, Size{W: 3, H: 2}); len(got) != 1 || got[0].Rect != (Rect{0, 0, 3, 2}) {
		t.Errorf("leaf rects = %+v", got)
	}
	if got := Rects(Column(1), Size{W: 3, H: 2}); len(got) != 1 {
		t.Errorf("empty column rects = %+v", got)
	}
	if got := Rects(GridNode(nil, 0), Size{W: 3, H: 2}); len(got) != 1 {
		t.Errorf("empty grid rects = %+v", got)
	}
	if got := Rects(fill{'a', 1, 1}, Size{}); len(got) != 1 || !got[0].Rect.Empty() {
		t.Errorf("zero-size rects = %+v", got)
	}
}

func TestRectContainsAndEmpty(t *testing.T) {
	r := Rect{2, 3, 4, 2}
	for _, c := range []struct {
		x, y int
		in   bool
	}{{2, 3, true}, {5, 4, true}, {6, 3, false}, {2, 5, false}, {1, 3, false}} {
		if r.Contains(c.x, c.y) != c.in {
			t.Errorf("Contains(%d,%d) = %v", c.x, c.y, !c.in)
		}
	}
	if !(Rect{}).Empty() || r.Empty() {
		t.Error("Empty wrong")
	}
}

// --- randomized: the rectangles are where Render draws the content.

type treeGen struct {
	rng  *rand.Rand
	next rune
}

func (g *treeGen) leaf() Node {
	g.next++
	return fill{ch: 0x100 + g.next, w: 1 + g.rng.Intn(6), h: 1 + g.rng.Intn(3)}
}

func (g *treeGen) flexChild(depth int) FlexChild {
	k := FlexChild{Node: g.node(depth)}
	switch g.rng.Intn(4) {
	case 0:
		k.Grow = 1 + g.rng.Intn(3)
	case 1:
		k.Shrink = 1 + g.rng.Intn(2)
	case 2:
		k.Basis = 1 + g.rng.Intn(5)
	}
	if g.rng.Intn(4) == 0 {
		k.Min = g.rng.Intn(3)
	}
	if g.rng.Intn(5) == 0 {
		k.Max = 1 + g.rng.Intn(6)
	}
	k.CrossAlign = CrossAlign(g.rng.Intn(4))
	return k
}

func (g *treeGen) node(depth int) Node {
	if depth <= 0 {
		return g.leaf()
	}
	switch g.rng.Intn(8) {
	case 0, 1:
		n := g.rng.Intn(4)
		kids := make([]FlexChild, n)
		for i := range kids {
			kids[i] = g.flexChild(depth - 1)
		}
		gap := g.rng.Intn(3) - 0
		if g.rng.Intn(2) == 0 {
			return RowJustify(gap, Justify(g.rng.Intn(6)), kids...)
		}
		return ColumnJustify(gap, Justify(g.rng.Intn(6)), kids...)
	case 2:
		box := NewBox().Padding(g.rng.Intn(2), g.rng.Intn(2), g.rng.Intn(2), g.rng.Intn(2))
		if g.rng.Intn(2) == 0 {
			box = box.Border(NormalBorder())
		}
		return BoxNode(box, g.node(depth-1))
	case 3:
		cols := 1 + g.rng.Intn(3)
		tracks := make([]Track, cols)
		for i := range tracks {
			switch g.rng.Intn(3) {
			case 0:
				tracks[i].Size = 1 + g.rng.Intn(4)
			case 1:
				tracks[i].Grow = 1
			}
		}
		cells := make([]Node, g.rng.Intn(2*cols+1))
		for i := range cells {
			cells[i] = g.node(depth - 1)
		}
		return GridNodeGaps(tracks, g.rng.Intn(3), g.rng.Intn(3), cells...)
	case 4:
		return Named("n", g.node(depth-1))
	case 5:
		return Fixed(g.node(depth-1), Size{W: g.rng.Intn(12) - 1, H: g.rng.Intn(8) - 1})
	case 6:
		return MinMax(g.node(depth-1), Size{W: g.rng.Intn(8) - 1, H: g.rng.Intn(5) - 1}, Size{W: g.rng.Intn(12) - 1, H: g.rng.Intn(8) - 1})
	}
	return g.leaf()
}

// cellsOf turns a rendered frame into rows of runes.
func cellsOf(out string) [][]rune {
	var rows [][]rune
	for _, l := range strings.Split(out, "\n") {
		rows = append(rows, []rune(l))
	}
	return rows
}

func TestRectsMatchWhereRenderDrawsEachLeaf(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for trial := 0; trial < 12000; trial++ {
		g := &treeGen{rng: rng}
		root := g.node(1 + rng.Intn(4))
		s := Size{W: 1 + rng.Intn(30), H: 1 + rng.Intn(14)}
		out := root.Render(s)
		cells := cellsOf(out)
		placed := Rects(root, s)

		// Every leaf marker appears exactly in its rectangle and nowhere else.
		for _, p := range placed {
			f, ok := p.Node.(fill)
			if !ok {
				continue
			}
			count := 0
			for y, row := range cells {
				for x, r := range row {
					if r != f.ch {
						continue
					}
					count++
					if !p.Rect.Contains(x, y) {
						t.Fatalf("trial %d: %q drawn at (%d,%d), outside its rect %+v\ntree size %v\n%s\n%#v", trial, f.ch, x, y, p.Rect, s, out, root)
					}
				}
			}
			if want := p.Rect.W * p.Rect.H; p.Rect.Empty() && count != 0 || !p.Rect.Empty() && count != want {
				t.Fatalf("trial %d: %q drawn in %d cells, its rect %+v covers %d\ntree size %v\n%s", trial, f.ch, count, p.Rect, want, s, out)
			}
		}

		// Every rectangle lies inside its parent's and the root's.
		stack := []Rect{}
		for _, p := range placed {
			stack = stack[:p.Depth]
			if p.Depth > 0 && !p.Rect.Empty() {
				par := stack[p.Depth-1]
				if p.Rect.X < par.X || p.Rect.Y < par.Y || p.Rect.X+p.Rect.W > par.X+par.W || p.Rect.Y+p.Rect.H > par.Y+par.H {
					t.Fatalf("trial %d: rect %+v is outside its parent %+v", trial, p.Rect, par)
				}
			}
			if p.Depth > 0 && p.Rect.Empty() {
				// An empty child keeps the parent's slot on the stack for its own children.
				stack = append(stack, stack[p.Depth-1])
				continue
			}
			stack = append(stack, p.Rect)
		}
		if placed[0].Rect != (Rect{0, 0, s.W, s.H}) {
			t.Fatalf("trial %d: root rect = %+v, want the full size %v", trial, placed[0].Rect, s)
		}
	}
}

// A grid row of height 0 (all its cells empty) adds no rows, so the rows after
// it start where Measure says. Regression: it rendered as one blank line.
func TestGridZeroHeightRowAddsNoRows(t *testing.T) {
	g := GridNode([]Track{{}, {}}, 1,
		Column(0), Column(0), // an all-empty first row
		Block("a"), Block("b"),
	)
	m := g.Measure(Unconstrained())
	if m.H != 2 { // 0 rows + 1 gap + 1 row
		t.Fatalf("Measure = %v, want height 2", m)
	}
	out := g.Render(m)
	if rows := strings.Split(out, "\n"); len(rows) != 2 || strings.TrimSpace(rows[0]) != "" || !strings.HasPrefix(rows[1], "a") {
		t.Errorf("Render = %q, want a blank gap row then the cells", out)
	}
	got := Rects(g, m)
	if got[len(got)-2].Rect.Y != 1 {
		t.Errorf("row 1 starts at y=%d, want 1", got[len(got)-2].Rect.Y)
	}
}
