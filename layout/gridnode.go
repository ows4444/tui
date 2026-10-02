package layout

// Track describes one grid column's width behaviour. The zero value sizes
// the column to its widest cell. Size > 0 fixes the width to exactly Size
// cells (Grow is then ignored). Otherwise Grow > 0 shares the grid's
// leftover width by weight, on top of the column's content width.
type Track struct {
	Size int
	Grow int
}

// GridNode arranges cells into len(tracks) columns, filling row by row; a
// short final row leaves its missing cells blank. gap cells separate columns
// and gap blank rows separate rows; use GridNodeGaps to give the two axes
// different gaps. Column widths come from the same solver as
// Row, so every row lines up; each row is as tall as its tallest cell,
// measured at that column's final width (so a cell can adapt its height to
// the width it is given), and every cell is stretched to its slot. Rendered
// at a Size it produces exactly that many columns and rows, padding spare
// space and clipping overflow like Row and Column. It replaces the string
// helpers Grid and GridFlex, removed in v1.0.
func GridNode(tracks []Track, gap int, cells ...Node) Node {
	return GridNodeGaps(tracks, gap, gap, cells...)
}

// GridNodeGaps is GridNode with separate gaps: colGap blank columns between
// columns and rowGap blank rows between rows. GridNodeGaps(t, 1, 0, cells...)
// is a table with one space between its columns and no blank line between its
// rows, which GridNode's single gap cannot express. A negative gap counts as
// zero. GridNode(tracks, g, cells...) is GridNodeGaps(tracks, g, g, cells...).
func GridNodeGaps(tracks []Track, colGap, rowGap int, cells ...Node) Node {
	return gridNode{tracks: tracks, colGap: colGap, rowGap: rowGap, cells: cells}
}

type gridNode struct {
	tracks []Track
	colGap int // blank columns between columns
	rowGap int // blank rows between rows
	cells  []Node
	ar     *arena // scratch for one Draw; nil outside Draw
}

func (g gridNode) cell(row, col int) Node {
	if i := row*len(g.tracks) + col; i < len(g.cells) {
		return g.cells[i]
	}
	return Block("")
}

func (g gridNode) rows() int {
	n := len(g.tracks)
	if n == 0 {
		return 0
	}
	return (len(g.cells) + n - 1) / n
}

// gapOf is the total of the n-1 gaps of size gap between n items.
func gapOf(gap, n int) int {
	if gap <= 0 || n < 2 {
		return 0
	}
	return gap * (n - 1)
}

// contentWidths returns each column's widest measured cell.
func (g gridNode) contentWidths() []int {
	w := intsFrom(g.ar, len(g.tracks))
	open := Constraints{MaxW: Unbounded, MaxH: Unbounded}
	for r := 0; r < g.rows(); r++ {
		for c := range g.tracks {
			if cw := g.cell(r, c).Measure(open).W; cw > w[c] {
				w[c] = cw
			}
		}
	}
	return w
}

func (g gridNode) rowHeights(widths []int) []int {
	h := intsFrom(g.ar, g.rows())
	for r := range h {
		for c := range g.tracks {
			m := g.cell(r, c).Measure(Constraints{MaxW: widths[c], MaxH: Unbounded})
			if m.H > h[r] {
				h[r] = m.H
			}
		}
	}
	return h
}

func sum(v []int) int {
	t := 0
	for _, x := range v {
		t += x
	}
	return t
}

func (g gridNode) Measure(c Constraints) Size {
	if len(g.tracks) == 0 || len(g.cells) == 0 {
		return c.Constrain(Size{})
	}
	widths := g.contentWidths()
	for i, t := range g.tracks {
		if t.Size > 0 {
			widths[i] = t.Size
		}
	}
	return c.Constrain(Size{
		W: sum(widths) + gapOf(g.colGap, len(widths)),
		H: sum(g.rowHeights(widths)) + gapOf(g.rowGap, g.rows()),
	})
}

// sizes solves the column widths at s.W and the row heights they produce.
func (g gridNode) sizes(s Size) (widths, heights []int) {
	content := g.contentWidths()
	specs := specsFrom(g.ar, len(g.tracks))
	for i, t := range g.tracks {
		if t.Size > 0 {
			specs[i] = flexSpec{basis: t.Size, min: t.Size, max: t.Size}
			continue
		}
		specs[i] = flexSpec{basis: content[i], grow: t.Grow, shrink: 1, max: Unbounded}
	}
	widths = solveFlexIn(g.ar, s.W-gapOf(g.colGap, len(g.tracks)), specs)
	return widths, g.rowHeights(widths)
}

func (g gridNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	if len(g.tracks) == 0 || len(g.cells) == 0 {
		return Block("").Render(s)
	}
	widths, heights := g.sizes(s)

	rows := make([]string, len(heights))
	for r := range rows {
		blocks := make([]string, len(g.tracks))
		for c := range g.tracks {
			blocks[c] = g.cell(r, c).Render(Size{W: widths[c], H: heights[r]})
		}
		rows[r] = joinHorizontal(g.colGap, blocks...)
	}
	// joinColumn, not joinVertical: a zero-height row must add no rows.
	return Block(joinColumn(g.rowGap, rows, heights)).Render(s)
}
