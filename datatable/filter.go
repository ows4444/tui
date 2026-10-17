package datatable

import (
	"sort"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/fuzzy"
	"github.com/ows4444/tui/keymap"
)

// Query returns the text the rows are filtered by; always empty when Filter
// is off.
func (m Model) Query() string { return m.query }

// Filtering reports whether the filter line is taking typed text. An app that
// quits on Esc or on a letter should leave those keys to the table while this
// is true.
func (m Model) Filtering() bool { return m.Filter && m.editing }

// SetQuery sets the filter text, keeping only the rows whose cells,
// joined with spaces, fuzzy-match it (see package fuzzy), in the order they
// had. The cursor goes to the first row. An empty query shows every row. It
// has no effect when Filter is off.
func (m *Model) SetQuery(q string) {
	if !m.Filter {
		return
	}
	m.query = q
	m.rebuild(false)
}

// SortBy orders the rows by column col, ascending or descending, whatever the
// current sort is. Numeric cells compare as numbers, others as strings, and
// the sort is stable. A column outside the headers is ignored.
func (m *Model) SortBy(col int, desc bool) {
	if m.Source != nil || col < 0 || col >= len(m.Headers) {
		return
	}
	m.sortCol, m.sortDesc, m.sorted = col, desc, true
	m.rebuild(true)
}

// ClearSort puts the rows back in the order they were given in (SetRows, or
// New), leaving any filter in place.
func (m *Model) ClearSort() {
	if !m.sorted {
		return
	}
	m.sorted = false
	m.rebuild(true)
}

// SourceRow returns the index, among the rows given to New or SetRows, of
// displayed row i, or -1 when i is out of range. It differs from i while the
// rows are sorted or filtered.
func (m Model) SourceRow(i int) int {
	if i < 0 || i >= m.rowCount() {
		return -1
	}
	if m.idx == nil {
		return i
	}
	return m.idx[i]
}

// top is the number of lines drawn above the table: the filter line while one
// is showing.
func (m Model) top() int {
	if m.Filter && (m.editing || m.query != "") {
		return 1
	}
	return 0
}

// filterLine is the line drawn above the table while top is 1.
func (m Model) filterLine() string {
	line := "/ " + m.query
	if m.editing {
		line += "_"
	}
	if m.query != "" {
		line += ansi.NewStyle().Faint().Render(" (" + strconv.Itoa(len(m.Rows)) + " of " + strconv.Itoa(len(m.source())) + ")")
	}
	return line
}

// source is the rows in the caller's order.
func (m Model) source() [][]string {
	if m.src != nil {
		return m.src
	}
	return m.Rows
}

// rebuild recomputes Rows from the source rows, the query and the sort. With
// follow the cursor stays on the row it was on, when it is still shown;
// otherwise it goes to the first row.
func (m *Model) rebuild(follow bool) {
	if m.src == nil && !m.sorted && m.query == "" {
		m.idx = nil
		return
	}
	base := m.source()
	was := -1
	if follow && m.cursor >= 0 && m.cursor < len(m.Rows) {
		was = m.SourceRow(m.cursor)
	}
	before := len(m.Rows)

	idx := make([]int, 0, len(base))
	for i, row := range base {
		if m.query == "" || m.rowMatches(row) {
			idx = append(idx, i)
		}
	}
	if m.sorted {
		c := m.sortCol
		cell := func(i int) string {
			if c < len(base[i]) {
				return base[i][c]
			}
			return ""
		}
		sort.SliceStable(idx, func(a, b int) bool {
			x, y := cell(idx[a]), cell(idx[b])
			if m.sortDesc {
				return lessCell(y, x)
			}
			return lessCell(x, y)
		})
	}

	if !m.sorted && m.query == "" {
		// Back to the caller's order: use their slice again, so the width
		// cache that is keyed on it still holds.
		m.Rows, m.src, m.idx = base, nil, nil
	} else {
		rows := make([][]string, len(idx))
		for i, j := range idx {
			rows[i] = base[j]
		}
		m.src, m.Rows, m.idx = base, rows, idx
		if before == len(rows) && len(rows) == len(base) {
			m.widths.rekey(m.Rows) // same cells, new order: the widths still hold
		}
	}

	m.cursor = 0
	if was >= 0 {
		if m.idx == nil {
			m.cursor = was
		} else {
			for i, j := range m.idx {
				if j == was {
					m.cursor = i
					break
				}
			}
		}
	}
	if len(m.Rows) == 0 {
		m.cursor = 0
	}
	m.offset = 0
	m.reveal()
}

// rowMatches reports whether the row's cells, cleaned and joined with spaces,
// fuzzy-match the query.
func (m Model) rowMatches(row []string) bool {
	cells := make([]string, len(row))
	for i, c := range row {
		cells[i] = ansi.Clean(m.Raw, c)
	}
	_, _, ok := fuzzy.Score(m.query, strings.Join(cells, " "))
	return ok
}

// lessCell orders two cells: as numbers when both parse, else as strings.
func lessCell(x, y string) bool {
	fx, ex := strconv.ParseFloat(x, 64)
	fy, ey := strconv.ParseFloat(y, 64)
	if ex == nil && ey == nil {
		return fx < fy
	}
	return x < y
}

// editFilter applies a key while the filter line is taking text, and reports
// whether it consumed the key. Up, Down and the page keys are left to the
// table so the rows can be walked while typing.
func (m *Model) editFilter(msg tui.Msg, k KeyMap) bool {
	key, ok := msg.(tui.Key)
	if !ok {
		return false
	}
	for _, b := range []keymap.Binding{k.Up, k.Down, k.PageUp, k.PageDown} {
		if keymap.Matches(msg, b) {
			return false
		}
	}
	switch {
	case key.Type == tui.KeyEsc:
		m.editing = false
		m.query = ""
		m.rebuild(false)
	case key.Type == tui.KeyEnter:
		m.editing = false
	case key.Type == tui.KeyBackspace:
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
			m.rebuild(false)
		}
	case key.Type == tui.KeySpace:
		m.query += " "
		m.rebuild(false)
	case key.Type == tui.KeyRunes && !key.Mod.Alt() && key.Text != "":
		m.query += key.Text
		m.rebuild(false)
	}
	return true
}
