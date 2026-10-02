package layout_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	. "github.com/ows4444/tui/layout"
)

// cellTree is a dashboard-shaped tree of CellNodes only.
func cellTree() Node {
	cells := make([]Node, 24)
	for i := range cells {
		cells[i] = Block("cell " + string(rune('a'+i%26)))
	}
	return Column(0,
		FlexChild{Node: BoxNode(NewBox().Border(NormalBorder()), Block("header"))},
		FlexChild{Grow: 1, Node: Row(1,
			FlexChild{Node: BoxNode(NewBox().Border(NormalBorder()), Block("nav\nnav\nnav")), Basis: 20},
			FlexChild{Grow: 1, Node: GridNode([]Track{{}, {Grow: 1}, {Size: 8}}, 1, cells...)},
		)},
		FlexChild{Node: RowJustify(2, JustifySpaceBetween,
			FlexChild{Node: Block("status")}, FlexChild{Node: Block("q: quit"), CrossAlign: CrossEnd})},
	)
}

// sameScreen draws root both ways under c and fails if the screens differ.
func sameScreen(t *testing.T, name string, root Node, c Constraints) {
	t.Helper()
	str := Draw(root, c)
	buf := cellbuf.New(c.MaxW, c.MaxH)
	if c.MaxW >= Unbounded || c.MaxH >= Unbounded {
		buf = cellbuf.New(200, 100)
	}
	sz := DrawTo(root, cellbuf.Layout(buf), c)
	want, err := cellbuf.Parse(str)
	if err != nil {
		t.Fatalf("%s: parse Draw output: %v", name, err)
	}
	if str == "" {
		if sz != (Size{}) {
			t.Errorf("%s: DrawTo size %v for empty Draw", name, sz)
		}
		return
	}
	if sz.W != want.Width() || sz.H != want.Height() {
		t.Fatalf("%s: DrawTo size %v, Draw %dx%d", name, sz, want.Width(), want.Height())
	}
	got := buf.Sub(cellbuf.Rect{W: sz.W, H: sz.H})
	if g, w := got.String(), want.String(); g != w {
		t.Errorf("%s: screens differ\n--- DrawTo\n%s\n--- Draw\n%s", name, g, w)
	}
	for y := 0; y < sz.H; y++ {
		for x := 0; x < sz.W; x++ {
			if got.At(x, y).Cluster != want.At(x, y).Cluster || got.At(x, y).Width != want.At(x, y).Width ||
				got.Style(got.At(x, y).Style) != want.Style(want.At(x, y).Style) {
				t.Fatalf("%s: cell (%d,%d) differs: %+v vs %+v", name, x, y, got.At(x, y), want.At(x, y))
			}
		}
	}
}

func TestDrawToMatchesDraw(t *testing.T) {
	red := ansi.NewStyle().Foreground(ansi.Red).Bold()
	long := "the quick brown fox jumps over the lazy dog and keeps running"
	cases := map[string]Node{
		"tree":    cellTree(),
		"text":    Text(long, WithWrap()),
		"styled":  StyledText(long, red, true),
		"colored": Block(red.Render("red\nline") + "\nplain"),
		"box":     BoxNode(NewBox().Border(RoundedBorder()).Padding(1, 2, 1, 2).Title("title", AlignCenter).BorderColor(ansi.Green), Text(long, WithWrap())),
		"boxbg":   BoxNode(NewBox().Border(NormalBorder()).Background(ansi.Blue), Block("x")),
		"margin":  BoxNode(NewBox().Border(NormalBorder()).Margin(1, 1, 1, 1), Block("x")),
		"sides":   BoxNode(NewBox().Border(NormalBorder()).BorderSides(true, false, true, true), Block("abc")),
		"scroll":  Scroll(Column(0, FlexChild{Node: Block("1\n2\n3\n4\n5\n6\n7\n8\n9")}), 3),
		"wide":    Row(1, FlexChild{Node: Block("日本語テキスト"), Shrink: 1}, FlexChild{Node: Block("abc")}),
		"overlay": OverlayNode(Column(0, FlexChild{Node: Block("aaaaaaaaaa\nbbbbbbbbbb\ncccccccccc")}), BoxNode(NewBox().Border(NormalBorder()), Block("hi")), 2, 1),
		"fixed":   Fixed(Block("fixed"), Size{W: 9, H: 2}),
		"mixed":   Column(1, FlexChild{Node: Fixed(Block("legacy"), Size{W: 4, H: 1})}, FlexChild{Node: Block("modern"), CrossAlign: CrossCenter}),
		"empty":   Block(""),
	}
	for name, n := range cases {
		for _, sz := range []Size{{120, 40}, {30, 8}, {7, 3}, {1, 1}} {
			sameScreen(t, fmt.Sprintf("%s %v", name, sz), n, Tight(sz))
		}
		sameScreen(t, name+" loose", n, Constraints{MaxW: 40, MaxH: 20})
	}
}

// randTree builds a random tree of CellNode containers over short leaves.
func randTree(r *rand.Rand, depth int) Node {
	if depth == 0 || r.Intn(4) == 0 {
		switch r.Intn(3) {
		case 0:
			return Block(fmt.Sprintf("leaf%d\nsecond line %d", r.Intn(99), r.Intn(9)))
		case 1:
			return Text("some words to wrap here", WithWrap())
		}
		return StyledText("styled", ansi.NewStyle().Italic(), false)
	}
	kids := make([]FlexChild, 1+r.Intn(3))
	for i := range kids {
		kids[i] = FlexChild{Node: randTree(r, depth-1), Grow: r.Intn(3), Shrink: r.Intn(2), Basis: r.Intn(8),
			CrossAlign: CrossAlign(r.Intn(4))}
	}
	switch r.Intn(5) {
	case 0:
		return Row(r.Intn(3), kids...)
	case 1:
		return Column(r.Intn(3), kids...)
	case 2:
		return BoxNode(NewBox().Border(NormalBorder()).Padding(r.Intn(2), r.Intn(2), r.Intn(2), r.Intn(2)), randTree(r, depth-1))
	case 3:
		cells := make([]Node, 2+r.Intn(5))
		for i := range cells {
			cells[i] = randTree(r, depth-1)
		}
		return GridNodeGaps([]Track{{}, {Grow: 1}, {Size: 5}}[:1+r.Intn(3)], r.Intn(2), r.Intn(2), cells...)
	}
	return Scroll(randTree(r, depth-1), r.Intn(5))
}

func TestDrawToMatchesDrawRandom(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	for i := 0; i < 300; i++ {
		n := randTree(r, 3)
		sz := Size{W: 1 + r.Intn(60), H: 1 + r.Intn(24)}
		sameScreen(t, fmt.Sprintf("tree %d %v", i, sz), n, Tight(sz))
	}
}

// BenchmarkNodeTreeCells is BenchmarkNodeTree drawn into a cell buffer: a
// tree of CellNodes only.
func BenchmarkNodeTreeCells(b *testing.B) {
	ui := cellTree()
	buf := cellbuf.New(120, 40)
	c := Constraints{MinW: 120, MaxW: 120, MinH: 40, MaxH: 40}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DrawTo(ui, cellbuf.Layout(buf), c)
	}
}

// TestDrawToAllocs pins acceptance criterion: a tree of CellNodes allocates at
// most 10 times per frame (the pool is unreliable under -race, so skip there).
func TestDrawToAllocs(t *testing.T) {
	if raceEnabled {
		t.Skip("sync.Pool drops items under the race detector")
	}
	ui := cellTree()
	buf := cellbuf.New(120, 40)
	c := Constraints{MinW: 120, MaxW: 120, MinH: 40, MaxH: 40}
	DrawTo(ui, cellbuf.Layout(buf), c) // warm the arena pool
	if n := testing.AllocsPerRun(50, func() { DrawTo(ui, cellbuf.Layout(buf), c) }); n > 10 {
		t.Errorf("DrawTo allocs/op = %v, want <= 10", n)
	}
}
