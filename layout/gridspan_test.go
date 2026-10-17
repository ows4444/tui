package layout

import (
	"strconv"
	"strings"
	"testing"
)

// spanGrid is a grid of tracks over cells, for looking at the placement.
func spanGrid(tracks []Track, cells ...Node) gridNode {
	return gridNode{tracks: tracks, cells: cells}.withSpans()
}

func lines(s string) []string { return strings.Split(s, "\n") }

func TestSpanWideCellOverTwo(t *testing.T) {
	g := GridNode([]Track{{Size: 10}, {Size: 10}}, 1, Span(Block("WIDE"), 2, 1), Block("L"), Block("R"))
	got := lines(g.Render(Size{W: 21, H: 3}))
	want := []string{
		"WIDE" + strings.Repeat(" ", 17),
		strings.Repeat(" ", 21),
		"L" + strings.Repeat(" ", 9) + " " + "R" + strings.Repeat(" ", 9),
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
}

func TestSpanTallCellBesideTwo(t *testing.T) {
	g := GridNode([]Track{{Size: 4}, {Size: 4}}, 0, Block("A"), Span(Block("T\nT"), 1, 2), Block("B"))
	got := g.Render(Size{W: 8, H: 2})
	if got != "A   T   \nB   T   " {
		t.Fatalf("got %q", got)
	}
}

func TestSpanWidensAutoColumns(t *testing.T) {
	g := GridNode([]Track{{}, {}}, 0, Span(Block("0123456789"), 2, 1), Block("a"), Block("b"))
	if w := g.Measure(Unconstrained()).W; w != 10 {
		t.Fatalf("Measure W = %d; the wide cell needs 10", w)
	}
	row := lines(g.Render(g.Measure(Unconstrained())))
	if i := strings.Index(row[1], "b"); i != 5 {
		t.Fatalf("second column starts at %d; want 5 (the 8 missing cells split 4 and 4): %q", i, row[1])
	}
}

func TestSpanDoesNotWidenFixedColumns(t *testing.T) {
	g := GridNode([]Track{{Size: 2}, {Size: 2}}, 0, Span(Block("0123456789"), 2, 1), Block("a"), Block("b"))
	if w := g.Measure(Unconstrained()).W; w != 4 {
		t.Fatalf("fixed columns grew to %d", w)
	}
}

func TestSpanGrowsRows(t *testing.T) {
	g := GridNode([]Track{{}, {}}, 0, Span(Block("1\n2\n3"), 1, 2), Block("x"), Block("y"))
	if h := g.Measure(Unconstrained()).H; h != 3 {
		t.Fatalf("Measure H = %d; the tall cell needs 3", h)
	}
	out := lines(g.Render(g.Measure(Unconstrained())))
	if len(out) != 3 || !strings.HasPrefix(out[0], "1") || !strings.HasPrefix(out[2], "3") {
		t.Fatalf("render:\n%v", out)
	}
}

func TestSpanPlacementSkipsCoveredPlaces(t *testing.T) {
	g := spanGrid([]Track{{}, {}, {}},
		Span(Block("a"), 1, 2), Block("b"), Block("c"), Block("d"), Block("e"))
	var got []string
	for _, s := range g.slots() {
		got = append(got, string(s.node.(block))+":"+strconv.Itoa(s.row)+","+strconv.Itoa(s.col))
	}
	if strings.Join(got, " ") != "a:0,0 b:0,1 c:0,2 d:1,1 e:1,2" {
		t.Fatalf("placement %v", got)
	}
}

func TestSpanWiderThanRemainingStartsNextRow(t *testing.T) {
	g := spanGrid([]Track{{}, {}, {}}, Block("a"), Span(Block("w"), 3, 1), Block("z"))
	var got []string
	for _, s := range g.slots() {
		got = append(got, string(s.node.(block))+":"+strconv.Itoa(s.row)+","+strconv.Itoa(s.col)+","+strconv.Itoa(s.cols))
	}
	if strings.Join(got, " ") != "a:0,0,1 w:1,0,3 z:2,0,1" {
		t.Fatalf("placement %v", got)
	}
	over := spanGrid([]Track{{}, {}}, Span(Block("w"), 9, 1))
	if s := over.slots(); len(s) != 1 || s[0].cols != 2 {
		t.Fatalf("an oversized span was not cut to the grid: %+v", s)
	}
}

func TestSpanOutsideAGridIsItsChild(t *testing.T) {
	n := Span(Block("hi"), 2, 3)
	if n.Render(Size{W: 3, H: 1}) != Block("hi").Render(Size{W: 3, H: 1}) {
		t.Fatal("Span changed how its child renders")
	}
	if Span(Span(Block("x"), 5, 5), 2, 1).(spanNode).cols != 2 {
		t.Fatal("an inner Span was not replaced")
	}
	if sp := Span(Block("x"), 0, -4).(spanNode); sp.cols != 1 || sp.rows != 1 {
		t.Fatal("a span under 1 is not 1")
	}
}

func TestSpanHitTestAndCells(t *testing.T) {
	g := GridNode([]Track{{Size: 10}, {Size: 10}}, 1,
		Span(Named("wide", Block("WIDE")), 2, 1), Named("l", Block("L")), Named("r", Block("R")))
	size := Size{W: 21, H: 3}
	for _, tt := range []struct {
		x, y int
		want string
	}{{0, 0, "wide"}, {20, 0, "wide"}, {0, 2, "l"}, {11, 2, "r"}} {
		if name, _, _, ok := NamedAt(g, size, tt.x, tt.y); !ok || name != tt.want {
			t.Errorf("NamedAt(%d,%d) = %q %v; want %q", tt.x, tt.y, name, ok, tt.want)
		}
	}
}

func TestSpanEmptyAndTiny(t *testing.T) {
	g := GridNode([]Track{{}, {}}, 1, Span(Block("x"), 2, 2))
	if g.Render(Size{}) != "" {
		t.Fatal("zero size must render nothing")
	}
	out := g.Render(Size{W: 1, H: 1})
	assertExact(t, out, Size{W: 1, H: 1})
}
