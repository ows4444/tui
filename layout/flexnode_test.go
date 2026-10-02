package layout

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func assertExact(t *testing.T, out string, s Size) {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != s.H {
		t.Fatalf("%d rows, want %d:\n%s", len(lines), s.H, out)
	}
	for i, l := range lines {
		if ansi.Width(l) != s.W {
			t.Fatalf("row %d is %d wide, want %d: %q", i, ansi.Width(l), s.W, l)
		}
	}
}

func TestRowRendersExactSizeAndGrowFillsMainAxis(t *testing.T) {
	r := Row(2,
		FlexChild{Node: Block("ab")},
		FlexChild{Node: Block("mid"), Grow: 1},
		FlexChild{Node: Block("z")},
	)
	s := Size{20, 3}
	out := r.Render(s)
	assertExact(t, out, s)
	first := strings.Split(out, "\n")[0]
	if !strings.HasPrefix(first, "ab  mid") || !strings.HasSuffix(first, "  z") {
		t.Errorf("row = %q", first)
	}
	// Preferred size: 2 + 3 + 1 + two gaps, one row tall.
	if got := r.Measure(Unconstrained()); got != (Size{2 + 3 + 1 + 4, 1}) {
		t.Errorf("Measure = %v", got)
	}
}

func TestRowNoGrowLeavesBlankSlackAndStaysExact(t *testing.T) {
	out := Row(0, FlexChild{Node: Block("ab")}).Render(Size{6, 2})
	assertExact(t, out, Size{6, 2})
}

func TestRowGrowSharesSlackByWeight(t *testing.T) {
	r := Row(0,
		FlexChild{Node: Block("a"), Grow: 1},
		FlexChild{Node: Block("b"), Grow: 3},
	)
	out := r.Render(Size{40, 1})
	assertExact(t, out, Size{40, 1})
	// Basis 1+1, 38 spare split 1:3 -> 9.5/28.5; "b" starts after a's
	// (1+9 or 1+10) columns.
	idx := strings.Index(out, "b")
	if idx < 10 || idx > 11 {
		t.Errorf("second child starts at column %d, want 10 or 11 (%q)", idx, out)
	}
}

func TestRowGrowRespectsMinMax(t *testing.T) {
	r := Row(0,
		FlexChild{Node: Block("a"), Grow: 1, Max: 5},
		FlexChild{Node: Block("b"), Grow: 1},
	)
	out := r.Render(Size{20, 1})
	if idx := strings.Index(out, "b"); idx != 5 {
		t.Errorf("b at %d, want 5 (a capped at Max 5): %q", idx, out)
	}
}

func TestRowShrinksThenTruncatesWithoutSplittingRuneOrEscape(t *testing.T) {
	styled := "\x1b[31mabcdefgh\x1b[0m"
	r := Row(0,
		FlexChild{Node: Block(styled), Shrink: 1},
		FlexChild{Node: Block("你好世界"), Shrink: 1},
	)
	s := Size{9, 1}
	out := r.Render(s)
	assertExact(t, out, s)
	if strings.Count(out, "\x1b[") != strings.Count(out, "\x1b[0m")+strings.Count(out, "\x1b[31m") {
		t.Errorf("escape sequence damaged: %q", out)
	}
	// Overflow that nothing can shrink is clipped, still exact.
	out = Row(0, FlexChild{Node: Block("abcdefgh")}).Render(Size{3, 1})
	assertExact(t, out, Size{3, 1})
	if !strings.HasPrefix(out, "abc") {
		t.Errorf("clip = %q", out)
	}
}

func TestShrinkRespectsMin(t *testing.T) {
	r := Row(0,
		FlexChild{Node: Block("aaaaaaaa"), Shrink: 1, Min: 6},
		FlexChild{Node: Block("bbbb"), Shrink: 1},
	)
	out := r.Render(Size{8, 1})
	if !strings.HasPrefix(out, "aaaaaa") || strings.Count(out, "b") != 2 {
		t.Errorf("out = %q, want 6 a's then 2 b's", out)
	}
}

func TestColumnStacksWithGapAndGrowsHeight(t *testing.T) {
	c := Column(1,
		FlexChild{Node: Block("top")},
		FlexChild{Node: Block("fill"), Grow: 1},
		FlexChild{Node: Block("bot")},
	)
	s := Size{6, 9}
	out := c.Render(s)
	assertExact(t, out, s)
	lines := strings.Split(out, "\n")
	if !strings.HasPrefix(lines[0], "top") || !strings.HasPrefix(lines[2], "fill") || !strings.HasPrefix(lines[8], "bot") {
		t.Errorf("column = %q", lines)
	}
	if got := c.Measure(Unconstrained()); got != (Size{4, 3 + 2}) {
		t.Errorf("Measure = %v", got)
	}
}

func TestEmptyAndZeroSize(t *testing.T) {
	if got := Row(1).Measure(Unconstrained()); got != (Size{}) {
		t.Errorf("empty Row Measure = %v", got)
	}
	assertExact(t, Row(1).Render(Size{3, 2}), Size{3, 2})
	assertExact(t, Column(1).Render(Size{3, 2}), Size{3, 2})
	if Row(0, FlexChild{Node: Block("x")}).Render(Size{0, 5}) != "" {
		t.Error("zero-width Render should be empty")
	}
}

func TestBoxNodeAddsChromeAndFillsSize(t *testing.T) {
	b := BoxNode(NewBox().Border(NormalBorder()).PaddingAll(1), Block("hi"))
	if got := b.Measure(Unconstrained()); got != (Size{2 + 4, 1 + 4}) {
		t.Errorf("Measure = %v, want {6 5}", got)
	}
	s := Size{12, 7}
	out := b.Render(s)
	assertExact(t, out, s)
	if !strings.HasPrefix(out, "┌") || !strings.HasSuffix(out, "┘") {
		t.Errorf("border missing:\n%s", out)
	}
	// Too small for its own chrome: still exact, no panic.
	assertExact(t, b.Render(Size{3, 2}), Size{3, 2})
	// Constrained measure never exceeds the bound.
	if got := b.Measure(Loose(Size{4, 4})); got.W > 4 || got.H > 4 {
		t.Errorf("constrained Measure = %v", got)
	}
}

func TestNestedLayoutRendersExact(t *testing.T) {
	ui := Column(0,
		FlexChild{Node: BoxNode(NewBox().Border(RoundedBorder()), Block("header"))},
		FlexChild{Grow: 1, Node: Row(1,
			FlexChild{Node: Block("nav"), Basis: 8},
			FlexChild{Node: BoxNode(NewBox().Border(NormalBorder()), Block("body")), Grow: 1},
		)},
	)
	assertExact(t, Draw(ui, Constraints{MinW: 40, MaxW: 40, MinH: 12, MaxH: 12}), Size{40, 12})
}

// randNode builds a random tree of Blocks, Rows, Columns and Boxes.
func randNode(r *rand.Rand, depth int) Node {
	if depth <= 0 || r.Intn(4) == 0 {
		words := []string{"", "x", "hello", "你好", "\x1b[1mbold\x1b[0m", "a\nbb\nccc"}
		return Block(words[r.Intn(len(words))])
	}
	if r.Intn(8) == 0 {
		return Fixed(randNode(r, depth-1), Size{W: r.Intn(12) - 1, H: r.Intn(8) - 1})
	}
	if r.Intn(8) == 0 {
		return MinMax(randNode(r, depth-1), Size{W: r.Intn(8) - 1, H: r.Intn(5) - 1}, Size{W: r.Intn(12) - 1, H: r.Intn(8) - 1})
	}
	if r.Intn(4) == 0 {
		return BoxNode(NewBox().Padding(r.Intn(2), r.Intn(2), r.Intn(2), r.Intn(2)).Border([]Border{{}, NormalBorder()}[r.Intn(2)]), randNode(r, depth-1))
	}
	if r.Intn(5) == 0 {
		tracks := make([]Track, 1+r.Intn(3))
		for i := range tracks {
			tracks[i] = Track{Size: r.Intn(4), Grow: r.Intn(3)}
		}
		cells := make([]Node, r.Intn(7))
		for i := range cells {
			cells[i] = randNode(r, depth-1)
		}
		// Independent gaps (equal ones included): GridNode(g) is GridNodeGaps(g, g).
		return GridNodeGaps(tracks, r.Intn(3), r.Intn(3), cells...)
	}
	kids := make([]FlexChild, r.Intn(4))
	for i := range kids {
		kids[i] = FlexChild{Node: randNode(r, depth-1), Basis: r.Intn(6), Grow: r.Intn(3), Shrink: r.Intn(3), Min: r.Intn(3), Max: r.Intn(10), CrossAlign: CrossAlign(r.Intn(4))}
	}
	j := Justify(r.Intn(6))
	if r.Intn(2) == 0 {
		return RowJustify(r.Intn(3), j, kids...)
	}
	return ColumnJustify(r.Intn(3), j, kids...)
}

func TestRandomTreesAlwaysRenderExactlyAndMeasureWithinBounds(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 3000; i++ {
		n := randNode(r, 4)
		c := Constraints{MinW: r.Intn(5), MaxW: r.Intn(40), MinH: r.Intn(4), MaxH: r.Intn(15)}
		got := n.Measure(c)
		want := c.Constrain(got)
		if got != want {
			t.Fatalf("tree %d: Measure %v escapes constraints %+v", i, got, c)
		}
		s := Size{W: r.Intn(40) - 2, H: r.Intn(15) - 2}
		out := n.Render(s)
		if s.W <= 0 || s.H <= 0 {
			if out != "" {
				t.Fatalf("tree %d: Render(%v) = %q, want empty", i, s, out)
			}
			continue
		}
		assertExact(t, out, s)
	}
}

func TestCrossAlignPlacesChildWithinTheCrossAxis(t *testing.T) {
	tall := Block("a\nb\nc")
	short := Block("X")
	row := func(a CrossAlign) []string {
		r := Row(1, FlexChild{Node: tall}, FlexChild{Node: short, CrossAlign: a})
		out := r.Render(Size{3, 5})
		assertExact(t, out, Size{3, 5})
		lines := strings.Split(out, "\n")
		col := make([]string, len(lines))
		for i, l := range lines {
			col[i] = string([]rune(l)[2]) // the short child's column
		}
		return col
	}
	for _, tc := range []struct {
		align CrossAlign
		want  string
	}{
		{CrossStretch, "X    "},
		{CrossStart, "X    "},
		{CrossCenter, "  X  "},
		{CrossEnd, "    X"},
	} {
		if got := strings.Join(row(tc.align), ""); got != tc.want {
			t.Errorf("align %d: short child column = %q, want %q", tc.align, got, tc.want)
		}
	}

	// In a Column, cross is horizontal.
	col := func(a CrossAlign) string {
		c := Column(0, FlexChild{Node: Block("wide-line")}, FlexChild{Node: Block("ab"), CrossAlign: a})
		out := c.Render(Size{9, 2})
		assertExact(t, out, Size{9, 2})
		return strings.Split(out, "\n")[1]
	}
	for a, want := range map[CrossAlign]string{CrossStretch: "ab       ", CrossStart: "ab       ", CrossCenter: "   ab    ", CrossEnd: "       ab"} {
		if got := col(a); got != want {
			t.Errorf("column align %d: %q, want %q", a, got, want)
		}
	}
}

func TestRowJustifyDistributesLeftoverSpace(t *testing.T) {
	kids := []FlexChild{{Node: Block("aa")}, {Node: Block("bb")}, {Node: Block("cc")}}
	cases := map[Justify]string{
		JustifyStart:        "aabbcc              ",
		JustifyEnd:          "              aabbcc",
		JustifyCenter:       "       aabbcc       ",
		JustifySpaceBetween: "aa       bb       cc",
		JustifySpaceAround:  "  aa    bb    cc    ",
		JustifySpaceEvenly:  "   aa   bb   cc   ",
	}
	for j, want := range cases {
		out := RowJustify(0, j, kids...).Render(Size{20, 1})
		assertExact(t, out, Size{20, 1})
		if strings.TrimRight(out, " ") != strings.TrimRight(want, " ") {
			t.Errorf("justify %d:\n got %q\nwant %q", j, out, want)
		}
	}
}

func TestColumnJustifyCentersVertically(t *testing.T) {
	out := ColumnJustify(0, JustifyCenter, FlexChild{Node: Block("x")}, FlexChild{Node: Block("y")}).Render(Size{1, 6})
	assertExact(t, out, Size{1, 6})
	if got := strings.ReplaceAll(out, "\n", "|"); got != " | |x|y| | " {
		t.Errorf("column = %q", got)
	}
}

func TestJustifyIsIgnoredWhenAChildGrows(t *testing.T) {
	plain := Row(0, FlexChild{Node: Block("a")}, FlexChild{Node: Block("b"), Grow: 1}).Render(Size{10, 1})
	just := RowJustify(0, JustifyEnd, FlexChild{Node: Block("a")}, FlexChild{Node: Block("b"), Grow: 1}).Render(Size{10, 1})
	if plain != just {
		t.Errorf("a growing child leaves no space to justify:\n plain %q\n just  %q", plain, just)
	}
}

func tall(n int) Node {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = "log line"
	}
	return Block(strings.Join(lines, "\n"))
}

func TestFillTakesLeftoverSpaceWhateverItsContentMeasures(t *testing.T) {
	// The middle child measures to 200 rows; the screen has 10. Header and
	// footer must stay fully visible and the middle gets exactly the rest.
	ui := Column(0,
		FlexChild{Node: Block("HEADER")},
		Fill(tall(200)),
		FlexChild{Node: Block("FOOTER")},
	)
	out := ui.Render(Size{10, 10})
	assertExact(t, out, Size{10, 10})
	lines := strings.Split(out, "\n")
	if !strings.HasPrefix(lines[0], "HEADER") || !strings.HasPrefix(lines[9], "FOOTER") {
		t.Errorf("siblings pushed out of view:\n%s", out)
	}
	for _, l := range lines[1:9] {
		if !strings.HasPrefix(l, "log line") {
			t.Errorf("middle should fill rows 1-8, got %q", l)
		}
	}
	// Without Fill the same tree loses its footer: the trap Fill exists for.
	bad := Column(0, FlexChild{Node: Block("HEADER")}, FlexChild{Node: tall(200), Grow: 1}, FlexChild{Node: Block("FOOTER")})
	if strings.Contains(bad.Render(Size{10, 10}), "FOOTER") {
		t.Error("expected the plain Grow child to push the footer out (documents why Fill exists)")
	}
}

func TestFillChildrenSplitByWeight(t *testing.T) {
	ui := Column(0, Fill(tall(50)), FillWeight(tall(50), 3), FillWeight(tall(50), 0))
	// weights 1, 3, and 0->1: 20 rows split 4 / 12 / 4.
	sizes := []int{}
	for _, k := range []FlexChild{Fill(nil), FillWeight(nil, 3), FillWeight(nil, 0)} {
		sizes = append(sizes, k.Grow)
	}
	if sizes[0] != 1 || sizes[1] != 3 || sizes[2] != 1 {
		t.Errorf("weights = %v, want [1 3 1]", sizes)
	}
	out := ui.Render(Size{8, 20})
	assertExact(t, out, Size{8, 20})
	got := solveFlex(20, []flexSpec{
		{basis: 1, grow: 1, shrink: 1, max: Unbounded},
		{basis: 1, grow: 3, shrink: 1, max: Unbounded},
		{basis: 1, grow: 1, shrink: 1, max: Unbounded},
	})
	if got[0]+got[1]+got[2] != 20 || got[1] < 3*got[0]-3 {
		t.Errorf("sizes = %v", got)
	}
}

func TestFillShrinksWhenTheContainerIsTooSmall(t *testing.T) {
	ui := Row(0, FlexChild{Node: Block("abcdef")}, Fill(tall(3)))
	out := ui.Render(Size{4, 3})
	assertExact(t, out, Size{4, 3})
}

// A child that renders at zero size contributes no rows or columns; only the
// gaps around it remain. Regression: a zero-height Column child rendered as
// "", which joinVertical counts as one blank line, so Render exceeded Measure
// and the last sibling was clipped.
func TestColumnZeroHeightChildContributesNoRows(t *testing.T) {
	kid := func(s string) FlexChild { return FlexChild{Node: Block(s)} }

	col := Column(0, kid("top"), kid(""), kid("bottom"))
	m := col.Measure(Unconstrained())
	if m != (Size{W: 6, H: 2}) {
		t.Fatalf("Measure = %v, want 6x2", m)
	}
	if got, want := col.Render(m), "top   \nbottom"; got != want {
		t.Errorf("Render(%v) = %q, want %q", m, got, want)
	}

	// The gaps around the empty child stay: top, gap, (nothing), gap, bottom.
	gapped := Column(1, kid("top"), kid(""), kid("bottom"))
	gm := gapped.Measure(Unconstrained())
	if gm.H != 4 {
		t.Fatalf("gapped Measure H = %d, want 4", gm.H)
	}
	if got, want := gapped.Render(gm), "top   \n      \n      \nbottom"; got != want {
		t.Errorf("gapped Render = %q, want %q", got, want)
	}

	// Leading and trailing empty children keep their gap rows too.
	edges := Column(1, kid(""), kid("x"), kid(""))
	em := edges.Measure(Unconstrained())
	if got, want := edges.Render(em), " \nx\n "; em.H != 3 || got != want {
		t.Errorf("edges Render(%v) = %q, want %q", em, got, want)
	}
}

// Draw's output is exactly the measured size for trees containing empty
// children, in both directions and inside a grid.
func TestEmptyChildrenKeepRenderAndMeasureConsistent(t *testing.T) {
	kid := func(s string) FlexChild { return FlexChild{Node: Block(s)} }
	trees := map[string]Node{
		"column":       Column(1, kid("a"), kid(""), kid("b"), kid("")),
		"row":          Row(2, kid("a"), kid(""), kid("b")),
		"nested":       Column(0, kid(""), FlexChild{Node: Row(1, kid(""), kid("x\ny"))}, kid("z")),
		"grid":         GridNode([]Track{{}, {}}, 1, Block("a"), Block(""), Block(""), Block("d")),
		"onlyEmpty":    Column(1, kid(""), kid("")),
		"columnNoKids": Column(1),
	}
	for name, n := range trees {
		m := n.Measure(Unconstrained())
		out := n.Render(m)
		if m.W == 0 || m.H == 0 {
			if out != "" {
				t.Errorf("%s: measured %v but rendered %q", name, m, out)
			}
			continue
		}
		rows := strings.Split(out, "\n")
		if len(rows) != m.H {
			t.Errorf("%s: %d rows, measured %d:\n%s", name, len(rows), m.H, out)
		}
		for i, r := range rows {
			if ansi.Width(r) != m.W {
				t.Errorf("%s row %d is %d wide, want %d: %q", name, i, ansi.Width(r), m.W, r)
			}
		}
	}
	// No sibling is clipped away.
	col := Column(0, kid("first"), kid(""), kid("last"))
	if out := Draw(col, Unconstrained()); !strings.Contains(out, "last") {
		t.Errorf("last sibling clipped: %q", out)
	}
}

// bigList is a Windowed child of n rows that records how many rows it was
// asked to render.
type bigList struct {
	n        int
	rendered int
}

func (b *bigList) Measure(c Constraints) Size { return c.Constrain(Size{W: 8, H: b.n}) }
func (b *bigList) Render(s Size) string       { b.rendered += s.H; return Block("full").Render(s) }
func (b *bigList) Rows(int) int               { return b.n }
func (b *bigList) RenderRows(w, start, count int) string {
	b.rendered += count
	rows := make([]string, count)
	for i := range rows {
		rows[i] = fmt.Sprintf("row%d", start+i)
	}
	return strings.Join(rows, "\n")
}

func TestScrollWindowedAsksForVisibleRowsOnly(t *testing.T) {
	b := &bigList{n: 100000}
	n := Scroll(b, 500)
	got := n.Render(Size{W: 8, H: 20})
	if b.rendered > 20 {
		t.Fatalf("child asked to render %d rows, want <= 20", b.rendered)
	}
	lines := strings.Split(got, "\n")
	if len(lines) != 20 || !strings.HasPrefix(lines[0], "row500") || !strings.HasPrefix(lines[19], "row519") {
		t.Fatalf("window wrong: first %q last %q (%d lines)", lines[0], lines[len(lines)-1], len(lines))
	}
	if h := n.Measure(Loose(Size{W: 8, H: 200000})).H; h != 100000 {
		t.Fatalf("Measure H = %d, want 100000", h)
	}
	// An offset past the end clamps to the last full window.
	got = Scroll(b, 1<<30).Render(Size{W: 8, H: 20})
	if !strings.HasPrefix(got, "row99980") {
		t.Fatalf("clamped window starts %q", got[:8])
	}
}
