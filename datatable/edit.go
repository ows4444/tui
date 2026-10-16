package datatable

import "slices"

// SetCell sets one cell of the source rows, the rows as given to New or
// SetRows (see SourceRow for the displayed row's index). It reports whether the
// row and column exist. The change is made on a copy, so the slices the caller
// passed in are not touched, a sort or filter in force is applied again, and
// the cached column widths are dropped.
func (m *Model) SetCell(row, col int, text string) bool {
	src := m.source()
	if row < 0 || row >= len(src) || col < 0 || col >= len(m.Headers) {
		return false
	}
	rows := slices.Clone(src)
	r := slices.Clone(rows[row])
	for len(r) <= col {
		r = append(r, "")
	}
	r[col] = text
	rows[row] = r
	m.SetRows(rows)
	return true
}

// InsertRow inserts cells as a new source row at index i, clamped to the
// rows there are (i at or past the end appends). It works on a copy, like
// SetCell, and applies a sort or filter in force.
func (m *Model) InsertRow(i int, cells []string) {
	src := m.source()
	i = max(0, min(i, len(src)))
	rows := make([][]string, 0, len(src)+1)
	rows = append(rows, src[:i]...)
	rows = append(rows, slices.Clone(cells))
	rows = append(rows, src[i:]...)
	m.SetRows(rows)
}

// RemoveRow removes the source row at index i and reports whether there was
// one. It works on a copy, like SetCell. The cursor keeps its displayed index,
// clamped to the rows that remain.
func (m *Model) RemoveRow(i int) bool {
	src := m.source()
	if i < 0 || i >= len(src) {
		return false
	}
	m.SetRows(slices.Delete(slices.Clone(src), i, i+1))
	return true
}
