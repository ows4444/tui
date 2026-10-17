package layout

import (
	"strings"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
)

// MergedRow is Row with no gap in which each child overlaps the next by one
// column, so two bordered boxes side by side share one border and the places
// where they meet become junctions: "┐" over "┌" is "┬", "┘" over "└" is "┴", and
// where one box is taller than its neighbour a "│" meets a "┌" as "├". Sizes are
// worked out as for Row (Basis, Grow, Shrink, Min and Max) on a row one column
// wider for each seam, so the children together fill exactly the width given.
//
// It is meant for children that draw a border on the side that meets their
// neighbour (a Box with a border). A seam is merged where both cells are line
// characters of the same family (thin, rounded and dashed, heavy, double, or
// ASCII); where one is blank the other shows; and in any other case the second
// child's cell shows, so a child without a border loses its edge column to the
// next. The merged cell takes the colour of the second child's.
func MergedRow(children ...FlexChild) Node {
	return mergedNode{inner: flexNode{row: true, kids: children}}
}

// MergedColumn is MergedRow for a column: each child overlaps the next by one
// row, and the bottom border of one box and the top border of the next become
// one row of junctions.
func MergedColumn(children ...FlexChild) Node {
	return mergedNode{inner: flexNode{kids: children}}
}

type mergedNode struct{ inner flexNode }

// seams is how many cells the children overlap in all.
func (m mergedNode) seams() int { return max(len(m.inner.kids)-1, 0) }

func (m mergedNode) Measure(c Constraints) Size {
	o := m.seams()
	cc := c
	if m.inner.row {
		cc.MaxW = growBound(cc.MaxW, o)
	} else {
		cc.MaxH = growBound(cc.MaxH, o)
	}
	main, cross := m.inner.split(m.inner.Measure(cc))
	return c.Constrain(m.inner.join(max(main-o, 0), cross))
}

// growBound is max widened by n, unless it is unbounded.
func growBound(max, n int) int {
	if max >= Unbounded {
		return max
	}
	return max + n
}

// plan sizes the children on a main axis one cell longer for each seam.
func (m mergedNode) plan(s Size) flexPlan {
	if m.inner.row {
		s.W += m.seams()
	} else {
		s.H += m.seams()
	}
	return m.inner.plan(s)
}

func (m mergedNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	pl := m.plan(s)
	var out []string // the lines built so far
	first := true
	for i, k := range m.inner.kids {
		size := pl.sizes[i]
		if size <= 0 {
			continue
		}
		lines := strings.Split(m.inner.renderChild(k, size, pl.crossSize), "\n")
		switch {
		case first:
			out, first = lines, false
		case m.inner.row:
			w := ansi.Width(out[0])
			for r := range out {
				if r < len(lines) {
					out[r] = joinSeam(out[r], w, lines[r])
				}
			}
		default:
			out[len(out)-1] = mergeLines(out[len(out)-1], lines[0])
			out = append(out, lines[1:]...)
		}
	}
	return Block(strings.Join(out, "\n")).Render(s)
}

// children places each child with its one-cell overlap, so Rects and hit
// testing agree with what Render draws.
func (m mergedNode) children(s Size) []child {
	if len(m.inner.kids) == 0 {
		return nil
	}
	pl := m.plan(s)
	out := make([]child, len(m.inner.kids))
	pos := 0
	for i, k := range m.inner.kids {
		off, cs := m.inner.crossPlacement(k, pl.sizes[i], pl.crossSize)
		if m.inner.row {
			out[i] = child{k.Node, Rect{X: pos, Y: off, W: pl.sizes[i], H: cs}}
		} else {
			out[i] = child{k.Node, Rect{X: off, Y: pos, W: cs, H: pl.sizes[i]}}
		}
		if pl.sizes[i] > 0 {
			pos += pl.sizes[i] - 1
		}
	}
	return out
}

// joinSeam appends b to a, which is w columns wide, with the last column of a
// and the first of b sharing one cell.
func joinSeam(a string, w int, b string) string {
	if w < 1 {
		return b
	}
	return ansi.Truncate(a, w-1) + mergeCell(ansi.TrimLeftWidth(a, w-1), ansi.Truncate(b, 1)) + ansi.TrimLeftWidth(b, 1)
}

// mergeCell merges two one-column styled strings into one.
func mergeCell(a, b string) string {
	_, ta := splitStyled(a)
	pb, tb := splitStyled(b)
	switch {
	case ta == " " || ta == "":
		return b
	case tb == " " || tb == "":
		return a
	}
	ra, na := utf8.DecodeRuneInString(ta)
	rb, nb := utf8.DecodeRuneInString(tb)
	if na != len(ta) || nb != len(tb) {
		return b
	}
	g, ok := mergeGlyph(ra, rb)
	if !ok {
		return b
	}
	if pb == "" {
		return string(g)
	}
	return pb + string(g) + ansi.Reset
}

// splitStyled returns the SGR sequences at the start of s and the text after
// them, with any later sequences removed.
func splitStyled(s string) (prefix, text string) {
	var pre, txt strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			if j >= len(s) {
				break
			}
			if s[j] == 'm' && txt.Len() == 0 {
				pre.WriteString(s[i : j+1])
			}
			i = j + 1
			continue
		}
		txt.WriteByte(s[i])
		i++
	}
	if pre.String() == ansi.Reset {
		return "", txt.String()
	}
	return pre.String(), txt.String()
}

// styledCell is one column of a line: its character and the SGR state it is
// drawn in.
type styledCell struct{ pre, text string }

// splitCells splits a line of one-column characters into cells. ok is false when
// the line holds anything else (a wide character, a link, a cursor movement),
// and then it is not merged.
func splitCells(line string) ([]styledCell, bool) {
	var cells []styledCell
	state := ""
	for i := 0; i < len(line); {
		if line[i] == 0x1b {
			if i+1 >= len(line) || line[i+1] != '[' {
				return nil, false
			}
			j := i + 2
			for j < len(line) && (line[j] < 0x40 || line[j] > 0x7e) {
				j++
			}
			if j >= len(line) || line[j] != 'm' {
				return nil, false
			}
			seq := line[i : j+1]
			if seq == ansi.Reset || seq == "\x1b[m" {
				state = ""
			} else {
				state += seq
			}
			i = j + 1
			continue
		}
		r, n := utf8.DecodeRuneInString(line[i:])
		if r == utf8.RuneError || ansi.Width(string(r)) != 1 {
			return nil, false
		}
		cells = append(cells, styledCell{pre: state, text: line[i : i+n]})
		i += n
	}
	return cells, true
}

// mergeLines merges two lines of the same width cell by cell: the lower border
// of one box and the upper border of the one below.
func mergeLines(a, b string) string {
	ca, oka := splitCells(a)
	cb, okb := splitCells(b)
	if !oka || !okb || len(ca) != len(cb) {
		return b
	}
	out := make([]styledCell, len(ca))
	for i := range ca {
		out[i] = mergeCellPair(ca[i], cb[i])
	}
	var sb strings.Builder
	prev := ""
	for _, c := range out {
		if c.pre != prev {
			if prev != "" {
				sb.WriteString(ansi.Reset)
			}
			sb.WriteString(c.pre)
			prev = c.pre
		}
		sb.WriteString(c.text)
	}
	if prev != "" {
		sb.WriteString(ansi.Reset)
	}
	return sb.String()
}

func mergeCellPair(a, b styledCell) styledCell {
	switch {
	case a.text == " ":
		return b
	case b.text == " ":
		return a
	}
	ra, _ := utf8.DecodeRuneInString(a.text)
	rb, _ := utf8.DecodeRuneInString(b.text)
	if g, ok := mergeGlyph(ra, rb); ok {
		return styledCell{pre: b.pre, text: string(g)}
	}
	return b
}
