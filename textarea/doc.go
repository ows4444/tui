package textarea

import (
	"slices"
	"sort"
)

// grpLines is how many lines one group holds (the last group may hold fewer).
const grpLines = 64

// dgroup is up to grpLines consecutive lines: their text, each line but the
// document's last followed by its '\n', and the start of each line inside text.
// A dgroup is immutable once built, so Model copies can share it.
//
// The text is a piece table: a list of chunks whose concatenation is the group's
// text. An edit copies only the chunk list and the few runes it inserts, and
// shares every other chunk with the old group, so an edit inside a very long line
// costs O(chunks), not O(line length). A fresh group has a single chunk.
type dgroup struct {
	chunks [][]rune
	cum    []int // cum[i] is the offset of chunks[i] within the group's text
	n      int   // total runes in the group
	ls     []int // ls[i] is the start of the group's i-th line within text; ls[0] == 0
}

const (
	maxPieces = 128 // chunk count at which a group is flattened back to one chunk
	mergeMax  = 256 // an insert is merged into the chunk before it up to this size
)

// newGroup returns a group over chunks; empty chunks are dropped. It takes
// ownership of the chunks slice (compacting it in place), so callers pass one
// they will not use again.
func newGroup(chunks [][]rune, ls []int) *dgroup {
	g := &dgroup{ls: ls}
	k := 0
	for _, c := range chunks {
		if len(c) > 0 {
			chunks[k] = c
			k++
		}
	}
	clear(chunks[k:])
	g.chunks = chunks[:k:k]
	g.cum = make([]int, k)
	for i, c := range g.chunks {
		g.cum[i] = g.n
		g.n += len(c)
	}
	return g
}

// chunkBefore returns the index of the last chunk starting before off (chunk 0
// for off == 0), the chunk whose end is at or after off.
func (g *dgroup) chunkBefore(off int) int {
	i := sort.Search(len(g.cum), func(i int) bool { return g.cum[i] >= off }) - 1
	return max(i, 0)
}

// at returns the rune at offset off.
func (g *dgroup) at(off int) rune {
	if len(g.chunks) == 1 {
		return g.chunks[0][off]
	}
	i := sort.Search(len(g.cum), func(i int) bool { return g.cum[i] > off }) - 1
	return g.chunks[i][off-g.cum[i]]
}

// slice returns the runes in [a, b) (b <= g.n); it aliases one chunk when it can.
func (g *dgroup) slice(a, b int) []rune {
	if a >= b {
		return nil
	}
	i := g.chunkBefore(a + 1)
	c := g.chunks[i]
	lo := a - g.cum[i]
	if b-g.cum[i] <= len(c) {
		return c[lo : b-g.cum[i] : b-g.cum[i]]
	}
	out := make([]rune, 0, b-a)
	out = append(out, c[lo:]...)
	for i++; i < len(g.chunks) && g.cum[i] < b; i++ {
		c = g.chunks[i]
		out = append(out, c[:min(len(c), b-g.cum[i])]...)
	}
	return out
}

// splice returns a new group with [off, offTo) replaced by ins and ls as given.
func (g *dgroup) splice(off, offTo int, ins []rune, ls []int) *dgroup {
	if g.n == 0 {
		return newGroup([][]rune{slices.Clone(ins)}, ls)
	}
	i, j := g.chunkBefore(off), g.chunkBefore(offTo)
	out := make([][]rune, 0, len(g.chunks)+2)
	out = append(out, g.chunks[:i]...)
	left := g.chunks[i][: off-g.cum[i] : off-g.cum[i]]
	right := g.chunks[j][offTo-g.cum[j]:]
	if len(ins) > 0 && len(left) > 0 && len(left)+len(ins) <= mergeMax {
		left = append(slices.Clone(left), ins...)
		ins = nil
	}
	out = append(out, left)
	if len(ins) > 0 {
		out = append(out, slices.Clone(ins))
	}
	out = append(out, right)
	out = append(out, g.chunks[j+1:]...)
	ng := newGroup(out, ls)
	if len(ng.chunks) > maxPieces {
		return newGroup([][]rune{ng.slice(0, ng.n)}, ls)
	}
	return ng
}

// doc is the text buffer together with its line index. It is a persistent
// structure: an edit inside one line returns a doc that shares every other group
// with the receiver, so a keystroke costs O(line + groups) instead of O(buffer),
// and Model copies holding an older doc are never disturbed. The zero doc is an
// empty buffer of one empty line.
type doc struct {
	gs     []*dgroup
	gstart []int // gstart[g] is the rune offset at which group g begins
	n      int   // total runes
	lines  int   // total lines, at least 1
}

// newDoc builds a doc holding a copy of rs.
func newDoc(rs []rune) doc {
	cp := slices.Clone(rs)
	d := doc{n: len(cp), lines: 1}
	start := 0
	ls := []int{0}
	closeGroup := func(end int) {
		d.gs = append(d.gs, newGroup([][]rune{cp[start:end:end]}, ls))
		d.gstart = append(d.gstart, start)
	}
	for i, r := range cp {
		if r != '\n' {
			continue
		}
		d.lines++
		if len(ls) == grpLines {
			closeGroup(i + 1)
			start = i + 1
			ls = []int{0}
		} else {
			ls = append(ls, i+1-start)
		}
	}
	closeGroup(len(cp))
	return d
}

func (d doc) len() int { return d.n }

func (d doc) lineCount() int { return max(d.lines, 1) }

// group returns the index of the group holding rune offset pos.
func (d doc) group(pos int) int {
	g := sort.Search(len(d.gstart), func(i int) bool { return d.gstart[i] > pos }) - 1
	return max(g, 0)
}

// lineStart returns the rune offset at which line l begins.
func (d doc) lineStart(l int) int {
	if len(d.gs) == 0 {
		return 0
	}
	g := l / grpLines
	return d.gstart[g] + d.gs[g].ls[l%grpLines]
}

// lineOf returns the index of the line containing rune offset pos.
func (d doc) lineOf(pos int) int {
	if len(d.gs) == 0 {
		return 0
	}
	g := d.group(pos)
	local := pos - d.gstart[g]
	ls := d.gs[g].ls
	i := sort.Search(len(ls), func(i int) bool { return ls[i] > local }) - 1
	return g*grpLines + max(i, 0)
}

// lineEnd returns the offset just past the text of line l, excluding its '\n'.
func (d doc) lineEnd(l int) int {
	if l+1 < d.lineCount() {
		return d.lineStart(l+1) - 1
	}
	return d.n
}

// at returns the rune at offset i.
func (d doc) at(i int) rune {
	g := d.group(i)
	return d.gs[g].at(i - d.gstart[g])
}

// slice returns the runes in [a, b). The result may alias the doc and must be
// treated as read-only.
func (d doc) slice(a, b int) []rune {
	if a >= b {
		return nil
	}
	g := d.group(a)
	base := d.gstart[g]
	gr := d.gs[g]
	if b-base <= gr.n {
		return gr.slice(a-base, b-base)
	}
	out := make([]rune, 0, b-a)
	out = append(out, gr.slice(a-base, gr.n)...)
	for g++; g < len(d.gs) && d.gstart[g] < b; g++ {
		out = append(out, d.gs[g].slice(0, min(d.gs[g].n, b-d.gstart[g]))...)
	}
	return out
}

func (d doc) str(a, b int) string { return string(d.slice(a, b)) }

// String returns the whole text.
func (d doc) String() string { return string(d.slice(0, d.n)) }

// replace returns the doc with [from, to) replaced by ins. When neither the
// removed nor the inserted text holds a newline the edit stays inside one line:
// only that line's group is rebuilt and later groups only move their start.
// Otherwise the doc is rebuilt from the flat text.
func (d doc) replace(from, to int, ins []rune) doc {
	removed := d.slice(from, to)
	if len(d.gs) == 0 || hasNewline(removed) || hasNewline(ins) {
		nv := make([]rune, 0, d.n-(to-from)+len(ins))
		nv = append(nv, d.slice(0, from)...)
		nv = append(nv, ins...)
		nv = append(nv, d.slice(to, d.n)...)
		return newDoc(nv)
	}
	delta := len(ins) - (to - from)
	l := d.lineOf(from)
	g, li := l/grpLines, l%grpLines
	old := d.gs[g]
	off, offTo := from-d.gstart[g], to-d.gstart[g]

	// Index slices are immutable once built, so an edit that moves nothing in
	// one shares it instead of copying.
	ls := old.ls
	if delta != 0 && li+1 < len(ls) {
		ls = slices.Clone(ls)
		for i := li + 1; i < len(ls); i++ {
			ls[i] += delta
		}
	}

	gs := slices.Clone(d.gs)
	gs[g] = old.splice(off, offTo, ins, ls)
	gstart := d.gstart
	if delta != 0 && g+1 < len(gstart) {
		gstart = slices.Clone(gstart)
		for i := g + 1; i < len(gstart); i++ {
			gstart[i] += delta
		}
	}
	return doc{gs: gs, gstart: gstart, n: d.n + delta, lines: d.lines}
}

func hasNewline(rs []rune) bool {
	for _, r := range rs {
		if r == '\n' {
			return true
		}
	}
	return false
}
