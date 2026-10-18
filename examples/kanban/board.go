package main

import "strings"

var columnNames = [3]string{"To do", "Doing", "Done"}

type card struct {
	id    int
	title string
}

// board is the data: three columns of cards and the next free card id. Every
// method returns a new board, so the model stays a value.
type board struct {
	cols   [3][]card
	nextID int
}

func sampleBoard() board {
	titles := [3][]string{
		{"Write the release notes", "Fix the flaky resize test", "Reply to the issue about wide characters", "Plan the v1 freeze"},
		{"Review the form groups pull request", "Record the demo for the README"},
		{"Cut rc.1", "Add the colour toolkit"},
	}
	var b board
	for c, ts := range titles {
		for _, t := range ts {
			b.nextID++
			b.cols[c] = append(b.cols[c], card{b.nextID, t})
		}
	}
	return b
}

func (b board) total() int { return len(b.cols[0]) + len(b.cols[1]) + len(b.cols[2]) }

func clone(cs []card) []card { return append([]card(nil), cs...) }

// add puts a new card at the end of column c and returns its row.
func (b board) add(c int, title string) (board, int) {
	b.nextID++
	b.cols[c] = append(clone(b.cols[c]), card{b.nextID, strings.TrimSpace(title)})
	return b, len(b.cols[c]) - 1
}

func (b board) rename(c, r int, title string) board {
	cs := clone(b.cols[c])
	cs[r].title = strings.TrimSpace(title)
	b.cols[c] = cs
	return b
}

func (b board) remove(c, r int) (board, card) {
	cs := clone(b.cols[c])
	gone := cs[r]
	b.cols[c] = append(cs[:r], cs[r+1:]...)
	return b, gone
}

// insert puts card k into column c at row r (clamped to the ends).
func (b board) insert(c, r int, k card) (board, int) {
	cs := clone(b.cols[c])
	r = max(0, min(r, len(cs)))
	cs = append(cs, card{})
	copy(cs[r+1:], cs[r:])
	cs[r] = k
	b.cols[c] = cs
	return b, r
}

// move takes the card at (c, r) to column to at row at, and returns where it
// landed.
func (b board) move(c, r, to, at int) (board, int) {
	b, k := b.remove(c, r)
	return b.insert(to, at, k)
}

// swap exchanges the card at (c, r) with its neighbour d rows away; it stays
// put at the ends.
func (b board) swap(c, r, d int) (board, int) {
	n := r + d
	if n < 0 || n >= len(b.cols[c]) {
		return b, r
	}
	cs := clone(b.cols[c])
	cs[r], cs[n] = cs[n], cs[r]
	b.cols[c] = cs
	return b, n
}
