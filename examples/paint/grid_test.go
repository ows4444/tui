package main

import (
	"strings"
	"testing"
)

func TestLineIsConnectedAndHitsBothEnds(t *testing.T) {
	g := newGrid(10, 6)
	g.line(pt{0, 0}, pt{9, 5}, 1, 1)
	if g.at(pt{0, 0}) != 1 || g.at(pt{9, 5}) != 1 {
		t.Fatal("ends not drawn")
	}
	for x := 0; x < 10; x++ {
		n := 0
		for y := 0; y < 6; y++ {
			if g.at(pt{x, y}) == 1 {
				n++
			}
		}
		if n == 0 {
			t.Fatalf("column %d has a gap", x)
		}
	}
}

func TestLineWorksInEveryDirection(t *testing.T) {
	for _, to := range []pt{{0, 4}, {4, 0}, {0, 0}, {4, 4}, {2, 4}} {
		g := newGrid(5, 5)
		g.line(pt{2, 2}, to, 1, 3)
		if g.at(pt{2, 2}) != 3 || g.at(to) != 3 {
			t.Errorf("line to %v misses an end", to)
		}
	}
}

func TestBrushSizeStampsASquare(t *testing.T) {
	g := newGrid(7, 7)
	g.stamp(pt{3, 3}, 3, 2)
	n := 0
	for _, c := range g.px {
		if c == 2 {
			n++
		}
	}
	if n != 9 || g.at(pt{2, 2}) != 2 || g.at(pt{4, 4}) != 2 {
		t.Fatalf("stamped %d pixels", n)
	}
	g = newGrid(3, 3)
	g.stamp(pt{0, 0}, 3, 1) // half off the edge
	if g.at(pt{0, 0}) != 1 || g.at(pt{1, 1}) != 1 {
		t.Fatal("a stamp at the edge was dropped")
	}
}

func TestRectOutlineAndFill(t *testing.T) {
	g := newGrid(8, 6)
	g.rect(pt{1, 1}, pt{6, 4}, 1, 4)
	if g.at(pt{1, 1}) != 4 || g.at(pt{6, 4}) != 4 || g.at(pt{3, 2}) != 0 {
		t.Fatalf("\n%s", g.text())
	}
	g.fill(pt{3, 2}, 5)
	if g.at(pt{3, 2}) != 5 || g.at(pt{0, 0}) != 0 {
		t.Fatalf("fill leaked or missed:\n%s", g.text())
	}
	g.fill(pt{0, 0}, 6)
	if g.at(pt{7, 5}) != 6 || g.at(pt{3, 2}) != 5 {
		t.Fatalf("outside fill wrong:\n%s", g.text())
	}
}

func TestTextCutsTrailingBlanks(t *testing.T) {
	g := newGrid(6, 4)
	g.stamp(pt{1, 0}, 1, 1)
	g.stamp(pt{2, 1}, 1, 7)
	if got, want := g.text(), " .\n  #\n"; got != want {
		t.Fatalf("%q, want %q", got, want)
	}
	if newGrid(3, 3).text() != "\n" {
		t.Fatalf("empty picture = %q", newGrid(3, 3).text())
	}
	if strings.ContainsAny(g.text(), "\t") {
		t.Fatal("tab in text")
	}
}

func TestResizeKeepsTheCornerAndCuts(t *testing.T) {
	g := newGrid(4, 4)
	g.stamp(pt{1, 1}, 1, 2)
	g.stamp(pt{3, 3}, 1, 3)
	big := g.resize(6, 5)
	if big.at(pt{1, 1}) != 2 || big.at(pt{3, 3}) != 3 || big.w != 6 {
		t.Fatal("growing lost pixels")
	}
	small := g.resize(2, 2)
	if small.at(pt{1, 1}) != 2 || len(small.px) != 4 {
		t.Fatal("shrinking is wrong")
	}
}
