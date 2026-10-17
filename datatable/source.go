package datatable

// Source is a table's rows read on demand, so a table over a database cursor,
// a log file or a million rows loads only the rows it draws. Rows is the
// number of rows and Row(i) the cells of row i, 0 <= i < Rows(). The table
// calls both from Update, View and DrawCells, in the program's goroutine, and
// calls Row only for the rows near the window, so Row should be cheap or
// cache; Rows is called often and should be O(1).
//
// A Source that changes (rows appended, a different query) should be replaced
// with a new NewSource, or the Model given the new count by SetSource; the
// table does not watch it. Row may return fewer cells than there are headers.
type Source interface {
	Rows() int
	Row(i int) []string
}

// NewSource builds a Model whose rows come from src. Natural column widths are
// measured from the rows in view only, so they can change while scrolling; set
// ColumnWidths for stable ones. Sorting, filtering and editing need all the
// rows in hand, so with a Source they are off: Sort, Filter, SortBy and the
// edit methods do nothing. Give the Model a Height (or let it see a
// tui.ResizeMsg): with neither, it reads at most defaultSourceRows rows.
// SetRows switches the Model back to in-memory rows.
func NewSource(headers []string, src Source) Model {
	m := New(headers, nil)
	m.Source = src
	return m
}

// SetSource replaces the Source, keeps the cursor at its index clamped to the
// new row count, and drops the cached widths.
func (m *Model) SetSource(src Source) {
	m.Source = src
	m.Rows, m.src, m.idx = nil, nil, nil
	m.sorted, m.query, m.editing = false, "", false
	m.widths = &widthCache{}
	m.SetCursor(m.cursor)
}

// defaultSourceRows bounds what a Source-backed table reads when it has no
// Height: the whole table is never loaded by accident.
const defaultSourceRows = 100

func (m Model) rowCount() int {
	if m.Source != nil {
		return max(m.Source.Rows(), 0)
	}
	return len(m.Rows)
}

func (m Model) rowAt(i int) []string {
	if m.Source != nil {
		return m.Source.Row(i)
	}
	return m.Rows[i]
}

// rowsIn returns rows from..to (to exclusive). With in-memory rows it is a
// sub-slice, not a copy; with a Source it is a fresh slice.
func (m Model) rowsIn(from, to int) [][]string {
	if m.Source == nil {
		return m.Rows[from:to]
	}
	out := make([][]string, to-from)
	for i := range out {
		out[i] = m.Source.Row(from + i)
	}
	return out
}
