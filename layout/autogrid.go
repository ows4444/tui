package layout

// AutoGrid lays cells out in as many equal columns as fit the width it is
// given: every column is at least minWidth wide, with gap blank columns and gap
// blank rows between cells. "As many 20-wide cards as fit" is
// AutoGrid(20, 1, cards...). At 45 columns that is two columns of 22 and 22, at
// 65 three of 21, 21 and 21; at less than minWidth it is one column. There are
// never more columns than cells, so two cards in a wide window stretch to fill
// it instead of leaving empty columns. Rows are as tall as their tallest cell,
// as in GridNode.
//
// With no width limit (Unbounded) the cells stand in one row, each minWidth
// wide. A minWidth under 1 counts as 1 and a negative gap as 0.
//
// It differs from Responsive, which swaps a whole layout at width breakpoints
// that you choose, and from GridNode, whose columns are a fixed list of Tracks.
func AutoGrid(minWidth, gap int, cells ...Node) Node {
	return AutoGridGaps(minWidth, gap, gap, cells...)
}

// AutoGridGaps is AutoGrid with separate gaps: colGap blank columns between
// columns and rowGap blank rows between rows.
func AutoGridGaps(minWidth, colGap, rowGap int, cells ...Node) Node {
	return autoGridNode{min: max(minWidth, 1), colGap: max(colGap, 0), rowGap: max(rowGap, 0), cells: append([]Node(nil), cells...)}
}

type autoGridNode struct {
	min, colGap, rowGap int
	cells               []Node
}

// columns is how many columns fit width w.
func (a autoGridNode) columns(w int) int {
	n := len(a.cells)
	if n == 0 {
		return 0
	}
	if w >= Unbounded {
		return n
	}
	return max(1, min(n, (w+a.colGap)/(a.min+a.colGap)))
}

// build is the GridNode for width w: equal columns that share w, the first few
// one cell wider when it does not divide.
func (a autoGridNode) build(w int) Node {
	cols := a.columns(w)
	if cols == 0 {
		return block("")
	}
	tracks := make([]Track, cols)
	if w >= Unbounded {
		for i := range tracks {
			tracks[i] = Track{Size: a.min}
		}
	} else {
		total := max(w-a.colGap*(cols-1), cols)
		for i := range tracks {
			tracks[i] = Track{Size: total / cols}
			if i < total%cols {
				tracks[i].Size++
			}
		}
	}
	return GridNodeGaps(tracks, a.colGap, a.rowGap, a.cells...)
}

func (a autoGridNode) Measure(c Constraints) Size {
	if c.MaxW <= 0 {
		return c.Constrain(Size{})
	}
	return c.Constrain(a.build(c.MaxW).Measure(c))
}

func (a autoGridNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	return place(a.build(s.W).Render(s), s.W, s.H, 0, 0)
}

func (a autoGridNode) children(s Size) []child {
	return []child{{node: a.build(s.W), rect: Rect{W: s.W, H: s.H}}}
}
