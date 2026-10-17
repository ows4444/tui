package layout

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

func boxed(text string, b Border) Node {
	return BoxNode(NewBox().Border(b), Block(text))
}

func mLines(s string) string { return strings.Join(strings.Split(ansi.StripANSI(s), "\n"), "|") }

func TestMergeGlyphJoinsLinesOfTheSameFamily(t *testing.T) {
	for _, tc := range []struct {
		a, b rune
		want rune
	}{
		{'┐', '┌', '┬'}, {'┘', '└', '┴'}, {'│', '│', '│'}, {'─', '─', '─'},
		{'│', '┌', '├'}, {'┤', '│', '┤'}, {'┐', '└', '┼'}, {'│', '─', '┼'},
		{'╮', '╭', '┬'}, {'╯', '╰', '┴'}, // rounded corners join as thin
		{'╌', '│', '┼'}, {'╎', '─', '┼'}, // dashes join as thin
		{'┓', '┏', '┳'}, {'┃', '┏', '┣'}, // heavy
		{'╗', '╔', '╦'}, {'╝', '╚', '╩'}, {'║', '╔', '╠'}, // double
		{'+', '+', '+'}, {'|', '-', '+'}, {'|', '|', '|'}, {'-', '-', '-'}, // ASCII
	} {
		if got, ok := mergeGlyph(tc.a, tc.b); !ok || got != tc.want {
			t.Errorf("merge %q %q = %q, %v; want %q", tc.a, tc.b, got, ok, tc.want)
		}
		if got, ok := mergeGlyph(tc.b, tc.a); !ok || got != tc.want {
			t.Errorf("merge is not symmetric for %q %q: %q, %v", tc.b, tc.a, got, ok)
		}
	}
	for _, tc := range [][2]rune{{'│', '║'}, {'┐', '╔'}, {'┃', '│'}, {'a', '│'}, {'│', ' '}, {'█', '█'}, {'+', '│'}} {
		if got, ok := mergeGlyph(tc[0], tc[1]); ok {
			t.Errorf("merge %q %q = %q; different families or non-lines must not merge", tc[0], tc[1], got)
		}
	}
}

func TestMergedRowSharesTheBorderBetweenBoxes(t *testing.T) {
	row := MergedRow(Fill(boxed("aaa", NormalBorder())), Fill(boxed("bbb", NormalBorder())))
	got := mLines(row.Render(Size{W: 11, H: 3}))
	want := "┌────┬────┐|│aaa │bbb │|└────┴────┘"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	three := MergedRow(Fill(boxed("a", RoundedBorder())), Fill(boxed("b", RoundedBorder())), Fill(boxed("c", RoundedBorder())))
	if got, want := mLines(three.Render(Size{W: 13, H: 3})), "╭───┬───┬───╮|│a  │b  │c  │|╰───┴───┴───╯"; got != want {
		t.Errorf("three boxes:\ngot  %s\nwant %s", got, want)
	}
}

func TestMergedColumnSharesTheBorderBetweenBoxes(t *testing.T) {
	col := MergedColumn(Fill(boxed("aaa", NormalBorder())), Fill(boxed("bbb", NormalBorder())))
	if got, want := mLines(col.Render(Size{W: 7, H: 5})), "┌─────┐|│aaa  │|├─────┤|│bbb  │|└─────┘"; got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestMergedBoxesNestedGiveTeesAndCrosses(t *testing.T) {
	right := MergedColumn(Fill(boxed("b", NormalBorder())), Fill(boxed("c", NormalBorder())))
	row := MergedRow(Fill(boxed("a", NormalBorder())), Fill(right))
	got := mLines(row.Render(Size{W: 9, H: 5}))
	want := "┌───┬───┐|│a  │b  │|│   ├───┤|│   │c  │|└───┴───┘"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	// A 2x2 grid meets in a cross.
	top := MergedRow(Fill(boxed("1", NormalBorder())), Fill(boxed("2", NormalBorder())))
	bot := MergedRow(Fill(boxed("3", NormalBorder())), Fill(boxed("4", NormalBorder())))
	grid := MergedColumn(Fill(top), Fill(bot))
	if got, want := mLines(grid.Render(Size{W: 9, H: 5})), "┌───┬───┐|│1  │2  │|├───┼───┤|│3  │4  │|└───┴───┘"; got != want {
		t.Errorf("grid:\ngot  %s\nwant %s", got, want)
	}
}

func TestMergedBordersOfDifferentFamiliesKeepTheSecond(t *testing.T) {
	row := MergedRow(Fill(boxed("a", NormalBorder())), Fill(boxed("b", DoubleBorder())))
	if got, want := mLines(row.Render(Size{W: 9, H: 3})), "┌───╔═══╗|│a  ║b  ║|└───╚═══╝"; got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	asc := MergedRow(Fill(boxed("a", ASCIIBorder())), Fill(boxed("b", ASCIIBorder())))
	got := strings.Split(ansi.StripANSI(asc.Render(Size{W: 9, H: 3})), "\n")
	if len(got) != 3 || got[0] != "+---+---+" || got[1] != "|a  |b  |" || got[2] != "+---+---+" {
		t.Errorf("ASCII: %q", got)
	}
}

func TestMergedCellKeepsTheSecondChildsColour(t *testing.T) {
	red := NewBox().Border(NormalBorder()).BorderColor(ansi.Red)
	green := NewBox().Border(NormalBorder()).BorderColor(ansi.Green)
	row := MergedRow(Fill(BoxNode(red, Block("a"))), Fill(BoxNode(green, Block("b"))))
	out := row.Render(Size{W: 9, H: 3})
	if !strings.Contains(out, ansi.NewStyle().Foreground(ansi.Green).Render("┬")) &&
		!strings.Contains(out, "\x1b[32m┬") {
		t.Errorf("the seam's junction is not in the second box's colour: %q", strings.Split(out, "\n")[0])
	}
	if got := mLines(out); got != "┌───┬───┐|│a  │b  │|└───┴───┘" {
		t.Errorf("text: %s", got)
	}
}

func TestMergedWidthsHonourBasisAndGrow(t *testing.T) {
	row := MergedRow(
		FlexChild{Node: boxed("a", NormalBorder()), Basis: 5},
		FlexChild{Node: boxed("b", NormalBorder()), Grow: 1, Basis: 3})
	got := mLines(row.Render(Size{W: 12, H: 3}))
	// 12 columns of picture are 13 on the row the children are planned on: 5 + 8.
	if want := "┌───┬──────┐|│a  │b     │|└───┴──────┘"; got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	rects := Rects(row, Size{W: 12, H: 3})
	var xs []Rect
	for _, p := range rects {
		if p.Depth == 1 {
			xs = append(xs, p.Rect)
		}
	}
	if len(xs) != 2 || xs[0] != (Rect{X: 0, Y: 0, W: 5, H: 3}) || xs[1] != (Rect{X: 4, Y: 0, W: 8, H: 3}) {
		t.Errorf("Rects = %+v, want boxes at x=0 (5 wide) and x=4 (8 wide)", xs)
	}
}

func TestMergedMeasureSubtractsTheSeams(t *testing.T) {
	a, b, c := boxed("aaa", NormalBorder()), boxed("bb", NormalBorder()), boxed("c", NormalBorder())
	wa := a.Measure(Constraints{MaxW: Unbounded, MaxH: Unbounded}).W
	wb := b.Measure(Constraints{MaxW: Unbounded, MaxH: Unbounded}).W
	wc := c.Measure(Constraints{MaxW: Unbounded, MaxH: Unbounded}).W
	got := MergedRow(FlexChild{Node: a}, FlexChild{Node: b}, FlexChild{Node: c}).Measure(Constraints{MaxW: Unbounded, MaxH: Unbounded})
	if want := wa + wb + wc - 2; got.W != want || got.H != 3 {
		t.Errorf("Measure = %+v, want %dx3", got, want)
	}
	col := MergedColumn(FlexChild{Node: a}, FlexChild{Node: b}).Measure(Constraints{MaxW: Unbounded, MaxH: Unbounded})
	if col.H != 5 {
		t.Errorf("column height = %d, want 5 (3 + 3 - 1)", col.H)
	}
	if got := MergedRow(FlexChild{Node: a}, FlexChild{Node: b}).Measure(Constraints{MaxW: 8, MaxH: 3}); got.W > 8 {
		t.Errorf("Measure ignored its bound: %+v", got)
	}
}

func TestMergedEdgeCases(t *testing.T) {
	if got := MergedRow().Render(Size{W: 5, H: 2}); strings.TrimSpace(got) != "" {
		t.Errorf("no children drew %q", got)
	}
	one := boxed("a", NormalBorder())
	if got, want := MergedRow(Fill(one)).Render(Size{W: 5, H: 3}), one.Render(Size{W: 5, H: 3}); got != want {
		t.Errorf("one child:\n%s\nwant\n%s", got, want)
	}
	if got := MergedRow(Fill(one), Fill(one)).Render(Size{W: 0, H: 3}); got != "" {
		t.Errorf("zero width drew %q", got)
	}
	// A child with no border loses its edge column to the next one.
	plain := MergedRow(Fill(Block("abcd")), Fill(boxed("x", NormalBorder())))
	if got := mLines(plain.Render(Size{W: 8, H: 3})); !strings.HasPrefix(got, "abc┌") {
		t.Errorf("borderless child: %s", got)
	}
	// A title with a wide character on the seam row: the row is not merged, and nothing breaks.
	wide := MergedColumn(Fill(BoxNode(NewBox().Border(NormalBorder()), Block("日本"))), Fill(boxed("b", NormalBorder())))
	if got := mLines(wide.Render(Size{W: 8, H: 6})); !strings.Contains(got, "日本") {
		t.Errorf("wide: %s", got)
	}
}

func TestSplitCellsHandlesStylesAndRefusesTheRest(t *testing.T) {
	cells, ok := splitCells("\x1b[31m┌\x1b[0m─\x1b[1;32m┐\x1b[0m")
	if !ok || len(cells) != 3 || cells[0].pre != "\x1b[31m" || cells[1].pre != "" || cells[2].pre != "\x1b[1;32m" {
		t.Errorf("cells = %+v, %v", cells, ok)
	}
	for _, bad := range []string{"日", "\x1b]8;;http://a\x1b\\x", "a\x1b[2Ab", "é"} {
		if _, ok := splitCells(bad); ok {
			t.Errorf("%q was split", bad)
		}
	}
}
