package datatable

import (
	"slices"
	"strings"
)

// Marked returns the source rows (see SourceRow) that are marked, ascending.
// Marks belong to the rows as given to New or SetRows, so they stay with a row
// through a sort or a filter.
func (m Model) Marked() []int {
	out := make([]int, 0, len(m.marked))
	for r := range m.marked {
		out = append(out, r)
	}
	slices.Sort(out)
	return out
}

// IsMarked reports whether source row r is marked.
func (m Model) IsMarked(r int) bool { return m.marked[r] }

// Mark sets whether source row r is marked. A row that does not exist is
// ignored. It works with Multi off too, so a program can mark rows itself.
func (m *Model) Mark(r int, on bool) {
	if r < 0 || (m.Source == nil && r >= len(m.source())) || (m.Source != nil && r >= m.rowCount()) {
		return
	}
	if m.marked[r] == on {
		return
	}
	next := make(map[int]bool, len(m.marked)+1)
	for k := range m.marked {
		next[k] = true
	}
	if on {
		next[r] = true
	} else {
		delete(next, r)
	}
	m.marked = next
}

// ClearMarks unmarks every row.
func (m *Model) ClearMarks() { m.marked = nil }

// toggleCursorMark flips the mark of the row under the cursor.
func (m *Model) toggleCursorMark() {
	r := m.SourceRow(m.cursor)
	m.Mark(r, !m.marked[r])
}

// toggleAllMarks marks every row shown, or unmarks them when all are marked.
func (m *Model) toggleAllMarks() {
	n := m.rowCount()
	all := n > 0
	for i := 0; i < n; i++ {
		if !m.marked[m.SourceRow(i)] {
			all = false
			break
		}
	}
	for i := 0; i < n; i++ {
		m.Mark(m.SourceRow(i), !all)
	}
}

// shiftMarks moves the marks after an insertion (delta +1 at row) or a removal
// (delta -1 at row, which loses its own mark).
func (m *Model) shiftMarks(row, delta int) {
	if len(m.marked) == 0 {
		return
	}
	next := make(map[int]bool, len(m.marked))
	for r := range m.marked {
		switch {
		case delta < 0 && r == row:
		case r >= row:
			next[r+delta] = true
		default:
			next[r] = true
		}
	}
	m.marked = next
}

// dropStaleMarks forgets marks past the end of new rows.
func (m *Model) dropStaleMarks(n int) {
	for r := range m.marked {
		if r >= n {
			next := make(map[int]bool, len(m.marked))
			for k := range m.marked {
				if k < n {
					next[k] = true
				}
			}
			m.marked = next
			return
		}
	}
}

// markGlyph is the mark drawn in the second gutter column.
func (m Model) markGlyph() string {
	if g := strings.TrimSpace(m.themed().GlyphSet().Check); g != "" {
		return g
	}
	return "*"
}

// markedRow reports whether displayed row i is marked.
func (m Model) markedRow(i int) bool { return len(m.marked) > 0 && m.marked[m.SourceRow(i)] }
