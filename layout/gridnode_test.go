package layout

import (
	"math/rand"
	"strings"
	"testing"
)

func TestGridNodeAlignsColumnsAcrossRows(t *testing.T) {
	g := GridNode([]Track{{}, {}}, 1,
		Block("a"), Block("longer"),
		Block("wide cell"), Block("b"),
	)
	out := g.Render(g.Measure(Unconstrained()))
	lines := strings.Split(out, "\n")
	// Column 2 starts at the same offset in both rows: 9 (widest) + gap 1.
	if i := strings.Index(lines[0], "longer"); i != 10 {
		t.Errorf("row 1 col 2 at %d, want 10: %q", i, lines[0])
	}
	if i := strings.Index(lines[2], "b"); i != 10 {
		t.Errorf("row 2 col 2 at %d, want 10: %q", i, lines[2])
	}
	assertExact(t, out, Size{16, 3})
}

func TestGridNodeTracksFixedGrowContent(t *testing.T) {
	g := GridNode([]Track{{}, {Grow: 1}, {Size: 4}}, 0, Block("x"), Block("y"), Block("z"))
	out := g.Render(Size{20, 1})
	assertExact(t, out, Size{20, 1})
	// col 1 content (1) + grow takes 20-1-4 = 15; col 3 fixed at 4.
	if strings.Index(out, "y") != 1 || strings.Index(out, "z") != 16 {
		t.Errorf("layout = %q", out)
	}
}

// wrapper adapts to the width it is given: its height depends on width.
type wrapper struct{ text string }

func (w wrapper) Measure(c Constraints) Size {
	width := c.MaxW
	if width <= 0 || width > len(w.text) {
		width = len(w.text)
	}
	return c.Constrain(Size{W: width, H: (len(w.text) + width - 1) / width})
}
func (w wrapper) Render(s Size) string {
	var rows []string
	for i := 0; i < len(w.text); i += s.W {
		end := min(i+s.W, len(w.text))
		rows = append(rows, w.text[i:end])
	}
	return Block(strings.Join(rows, "\n")).Render(s)
}

func TestGridNodeRowHeightFollowsFinalColumnWidth(t *testing.T) {
	g := GridNode([]Track{{Size: 4}, {Grow: 1}}, 0, Block("id"), wrapper{"abcdefgh"})
	// Column 2 is 6 wide at total width 10, so the 8-char text wraps to 2 rows.
	out := g.Render(Size{10, 2})
	assertExact(t, out, Size{10, 2})
	if !strings.Contains(out, "abcdef") || !strings.Contains(out, "gh") {
		t.Errorf("cell did not wrap to its allotted width:\n%s", out)
	}
}

func TestGridNodeShortLastRowAndDegenerate(t *testing.T) {
	g := GridNode([]Track{{}, {}}, 1, Block("a"), Block("b"), Block("c"))
	assertExact(t, g.Render(Size{9, 4}), Size{9, 4})
	if got := GridNode(nil, 1).Measure(Unconstrained()); got != (Size{}) {
		t.Errorf("no tracks Measure = %v", got)
	}
	assertExact(t, GridNode([]Track{{}}, 0).Render(Size{3, 2}), Size{3, 2})
	if GridNode([]Track{{}}, 0, Block("x")).Render(Size{0, 2}) != "" {
		t.Error("zero width should render empty")
	}
	// Overflow clips exactly.
	assertExact(t, GridNode([]Track{{}, {}}, 1, Block("abcdef"), Block("ghij")).Render(Size{5, 1}), Size{5, 1})
}

// GridNodeGaps with only a column gap separates columns and adds no blank
// rows between the rows.
func TestGridNodeGapsColumnGapOnly(t *testing.T) {
	g := GridNodeGaps([]Track{{}, {}}, 2, 0,
		Block("a"), Block("b"),
		Block("c"), Block("d"),
	)
	got := g.Render(g.Measure(Unconstrained()))
	if want := "a  b\nc  d"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if m := g.Measure(Unconstrained()); m != (Size{4, 2}) {
		t.Errorf("Measure = %v, want 4x2", m)
	}
}

// With only a row gap the rows are separated by blank rows and the columns
// touch.
func TestGridNodeGapsRowGapOnly(t *testing.T) {
	g := GridNodeGaps([]Track{{}, {}}, 0, 1,
		Block("a"), Block("b"),
		Block("c"), Block("d"),
	)
	got := g.Render(g.Measure(Unconstrained()))
	if want := "ab\n  \ncd"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if m := g.Measure(Unconstrained()); m != (Size{2, 3}) {
		t.Errorf("Measure = %v, want 2x3", m)
	}
}

// The two gaps are independent when both are set.
func TestGridNodeGapsBothGaps(t *testing.T) {
	g := GridNodeGaps([]Track{{}, {}}, 3, 2, Block("a"), Block("b"), Block("c"), Block("d"))
	lines := strings.Split(g.Render(g.Measure(Unconstrained())), "\n")
	if len(lines) != 4 || lines[0] != "a   b" || strings.TrimSpace(lines[1]) != "" || strings.TrimSpace(lines[2]) != "" || lines[3] != "c   d" {
		t.Errorf("lines = %q", lines)
	}
}

// Negative gaps behave as zero on their own axis, like Row and Column.
func TestGridNodeGapsNegativeGapIsZero(t *testing.T) {
	neg := GridNodeGaps([]Track{{}, {}}, -3, -1, Block("a"), Block("b"), Block("c"), Block("d"))
	zero := GridNodeGaps([]Track{{}, {}}, 0, 0, Block("a"), Block("b"), Block("c"), Block("d"))
	if neg.Render(Size{6, 4}) != zero.Render(Size{6, 4}) || neg.Measure(Unconstrained()) != zero.Measure(Unconstrained()) {
		t.Error("negative gaps differ from zero gaps")
	}
}

// GridNode(tracks, g, ...) is GridNodeGaps(tracks, g, g, ...): same Render,
// Measure and Rects for a spread of tracks, cells, gaps and sizes.
func TestGridNodeIsGridNodeGapsWithTheSameGap(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	words := []string{"", "a", "bb", "wide cell", "x\ny", "hello world here"}
	for i := 0; i < 400; i++ {
		tracks := make([]Track, 1+rng.Intn(4))
		for j := range tracks {
			switch rng.Intn(3) {
			case 0:
				tracks[j] = Track{Size: 1 + rng.Intn(6)}
			case 1:
				tracks[j] = Track{Grow: 1 + rng.Intn(3)}
			}
		}
		cells := make([]Node, rng.Intn(9))
		for j := range cells {
			cells[j] = Block(words[rng.Intn(len(words))])
		}
		gap := rng.Intn(4) - 1
		a, b := GridNode(tracks, gap, cells...), GridNodeGaps(tracks, gap, gap, cells...)
		if a.Measure(Unconstrained()) != b.Measure(Unconstrained()) {
			t.Fatalf("Measure differs (gap %d, tracks %v)", gap, tracks)
		}
		for _, s := range []Size{{W: 1, H: 1}, {W: 12, H: 5}, {W: 40, H: 10}, {W: 3, H: 9}} {
			if a.Render(s) != b.Render(s) {
				t.Fatalf("Render differs at %v (gap %d, tracks %v)", s, gap, tracks)
			}
			ra, rb := Rects(a, s), Rects(b, s)
			if len(ra) != len(rb) {
				t.Fatalf("Rects length differs at %v", s)
			}
			for k := range ra {
				if ra[k].Rect != rb[k].Rect || ra[k].Depth != rb[k].Depth {
					t.Fatalf("Rects[%d] differs at %v", k, s)
				}
			}
		}
	}
}

// GridNodeGaps renders exactly the size it is given and Measure matches the
// natural render, for independent gaps, including empty cells and short rows.
func TestGridNodeGapsRenderExactlyAndMeasureMatches(t *testing.T) {
	for _, gaps := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {2, 3}, {3, 1}} {
		g := GridNodeGaps([]Track{{}, {Size: 4}, {Grow: 1}}, gaps[0], gaps[1],
			Block("a"), Block(""), Block("c\nd"),
			Block(""), Block(""), Block(""),
			Block("wide cell"), Block("x"))
		natural := g.Measure(Unconstrained())
		assertExact(t, g.Render(natural), natural)
		for _, s := range []Size{{W: 1, H: 1}, {W: 9, H: 3}, {W: 30, H: 12}, {W: 300, H: 80}} {
			assertExact(t, g.Render(s), s)
		}
	}
}

// Each cell's rectangle is where its content is drawn, for independent gaps.
func TestGridNodeGapsRectsMatchRender(t *testing.T) {
	for _, gaps := range [][2]int{{0, 0}, {2, 0}, {0, 2}, {3, 1}} {
		cells := []Node{
			Named("a", Block("A")), Named("b", Block("B")),
			Named("c", Block("C")), Named("d", Block("D")),
		}
		g := GridNodeGaps([]Track{{Size: 2}, {Size: 3}}, gaps[0], gaps[1], cells...)
		s := g.Measure(Unconstrained())
		lines := strings.Split(g.Render(s), "\n")
		for _, name := range []string{"a", "b", "c", "d"} {
			r, ok := RectOf(g, s, name)
			if !ok {
				t.Fatalf("gaps %v: no rectangle for %s", gaps, name)
			}
			if r.Y >= len(lines) || r.X >= len(lines[r.Y]) || string(lines[r.Y][r.X]) != strings.ToUpper(name) {
				t.Errorf("gaps %v: %s's rectangle %+v does not point at its content in %q", gaps, name, r, lines)
			}
		}
	}
}
