package main

import "testing"

// board builds a game from rows: '*' is a mine, anything else is clear.
func board(rows ...string) game {
	g := newGame(level{"test", len(rows[0]), len(rows), 0}, 1)
	for y, r := range rows {
		for x, c := range r {
			if c == '*' {
				g.cells[y*g.lv.w+x].mine = true
				g.lv.mine++
			}
		}
	}
	g.left = g.lv.w*g.lv.h - g.lv.mine
	g.state = playing
	return g.count()
}

func TestCountsNeighbouringMines(t *testing.T) {
	g := board("*..", "...", "..*")
	for _, c := range []struct{ x, y, want int }{{1, 1, 2}, {1, 0, 1}, {2, 1, 1}, {0, 2, 0}} {
		if got := g.at(c.x, c.y).adj; got != c.want {
			t.Errorf("(%d,%d) = %d, want %d", c.x, c.y, got, c.want)
		}
	}
}

func TestFirstClickIsSafeAndOpensAnArea(t *testing.T) {
	for seed := uint64(1); seed < 40; seed++ {
		g := newGame(levels[0], seed).reveal(4, 4)
		if g.state != playing || g.at(4, 4).mine || g.at(4, 4).adj != 0 {
			t.Fatalf("seed %d: state %v mine %v adj %d", seed, g.state, g.at(4, 4).mine, g.at(4, 4).adj)
		}
		n := 0
		for _, c := range g.cells {
			if c.mine {
				n++
			}
		}
		if n != levels[0].mine {
			t.Fatalf("seed %d laid %d mines", seed, n)
		}
	}
}

func TestSameSeedLaysTheSameMines(t *testing.T) {
	a, b := newGame(levels[1], 9).reveal(3, 3), newGame(levels[1], 9).reveal(3, 3)
	for i := range a.cells {
		if a.cells[i].mine != b.cells[i].mine {
			t.Fatal("different mines")
		}
	}
}

func TestZeroFloodsAndStopsAtNumbers(t *testing.T) {
	g := board("....*", ".....", ".....").reveal(0, 2)
	if !g.at(0, 0).open || !g.at(2, 2).open || g.at(4, 0).open {
		t.Fatalf("flood wrong: %+v", g.cells)
	}
	if g.at(3, 0).open != true || g.at(3, 0).adj != 1 {
		t.Fatalf("the number edge did not open: %+v", g.at(3, 0))
	}
}

func TestOpeningAMineLosesAndShowsTheMines(t *testing.T) {
	g := board("*..", "..*", "...").reveal(0, 0)
	if g.state != lost || !g.at(0, 0).boom || !g.at(2, 1).open {
		t.Fatalf("state %v", g.state)
	}
	if again := g.reveal(1, 0); again.at(1, 0).open {
		t.Fatal("a lost game kept playing")
	}
}

func TestFlagsBlockOpeningAndCountDown(t *testing.T) {
	g := board("*..", "...").toggleFlag(0, 0)
	if g.minesLeft() != 0 || !g.at(0, 0).flag {
		t.Fatal("flag not set")
	}
	if g.reveal(0, 0).at(0, 0).open {
		t.Fatal("opened a flagged cell")
	}
	g = g.toggleFlag(0, 0)
	if g.minesLeft() != 1 || g.at(0, 0).flag {
		t.Fatal("flag not cleared")
	}
	if g = g.reveal(2, 1).toggleFlag(2, 1); g.at(2, 1).flag {
		t.Fatal("flagged an open cell")
	}
}

func TestChordOpensAroundAMatchingNumber(t *testing.T) {
	g := board("*..", "...", "...").reveal(1, 1)
	if g.at(0, 1).open {
		t.Fatal("a number opened its neighbours by itself")
	}
	if got := g.chord(1, 1); got.at(0, 1).open {
		t.Fatal("chord without a flag opened cells")
	}
	g = g.toggleFlag(0, 0).chord(1, 1)
	if !g.at(0, 1).open || !g.at(2, 2).open || g.state != won {
		t.Fatalf("chord did not clear the rest: state %v left %d", g.state, g.left)
	}
}

func TestChordOnAWrongFlagLoses(t *testing.T) {
	g := board("*..", "...", "...").reveal(1, 1).toggleFlag(1, 0).chord(1, 1)
	if g.state != lost {
		t.Fatalf("state %v", g.state)
	}
}

func TestOpeningEveryClearCellWins(t *testing.T) {
	g := board("*.", "..").reveal(1, 0).reveal(0, 1).reveal(1, 1)
	if g.state != won || g.left != 0 || !g.at(0, 0).flag {
		t.Fatalf("state %v left %d", g.state, g.left)
	}
}

func TestSmallBoardsStillGetASafeFirstCell(t *testing.T) {
	g := newGame(level{"tiny", 3, 3, 8}, 5).reveal(1, 1)
	if g.state != won || g.at(1, 1).mine {
		t.Fatalf("state %v", g.state)
	}
}
