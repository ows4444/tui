package virtuallist

import "strings"

// rowsOf is the number of rows item i takes when HeightOf is set.
func (m Model) rowsOf(i int) int { return max(m.HeightOf(i), 1) }

// shownCount is how many items, from the one at from, start inside a window of
// Height rows: the last may be cut off.
func (m Model) shownCount(from int) int {
	rows, n := 0, 0
	for i := max(from, 0); i < m.ItemCount && rows < m.Height; i++ {
		rows += m.rowsOf(i)
		n++
	}
	return n
}

// maxOffsetRows is the first offset from which every remaining item fits in the
// window, so scrolling stops with the last item at the bottom. An item taller
// than the window leaves its own index.
func (m Model) maxOffsetRows() int {
	rows, i := 0, m.ItemCount
	for i > 0 && rows+m.rowsOf(i-1) <= m.Height {
		i--
		rows += m.rowsOf(i)
	}
	if i == m.ItemCount && i > 0 {
		i--
	}
	return i
}

// followRows scrolls the least that shows the cursor item whole, or from its
// first row if it is taller than the window. It looks only at the items between
// the window and the cursor, from the cursor back, so a jump across a long list
// costs the window, not the distance.
func (m *Model) followRows() {
	if m.cursor < m.offset {
		m.setOffset(m.cursor)
		return
	}
	used, fits := 0, true
	for i := m.offset; i <= m.cursor; i++ {
		if used += m.rowsOf(i); used > m.Height {
			fits = false
			break
		}
	}
	if fits {
		return
	}
	rows, i := 0, m.cursor
	for i >= 0 && rows+m.rowsOf(i) <= m.Height {
		rows += m.rowsOf(i)
		i--
	}
	m.setOffset(i + 1)
	if m.offset > m.cursor {
		m.offset = m.cursor
	}
}

// itemAtRow is the item drawn on window row y, and false below the last.
func (m Model) itemAtRow(y int) (int, bool) {
	row := 0
	for i := max(m.offset, 0); i < m.ItemCount; i++ {
		row += m.rowsOf(i)
		if y < row {
			return i, true
		}
	}
	return 0, false
}

// viewRows is View when HeightOf is set: each item padded or cut to its rows,
// and the whole cut at Height.
func (m Model) viewRows() string {
	var rows []string
	for i := clamp(m.offset, 0, m.ItemCount); i < m.ItemCount && len(rows) < m.Height; i++ {
		lines := strings.Split(m.RenderItem(i), "\n")
		for k := 0; k < m.rowsOf(i) && len(rows) < m.Height; k++ {
			if k < len(lines) {
				rows = append(rows, strings.TrimSuffix(lines[k], "\r"))
			} else {
				rows = append(rows, "")
			}
		}
	}
	return strings.Join(rows, "\n")
}
