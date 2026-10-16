package hexview

import "github.com/ows4444/tui"

// wheelRows is how many rows one wheel notch scrolls.
const wheelRows = 3

// byteAt returns the offset of the byte under the cell at column x, row y of the
// area View draws (0,0 is its top-left cell), clamped to the data. A cell in the
// offset column is the first byte of its row; one in a gap, the byte before it.
func (m Model) byteAt(x, y int) int {
	if len(m.Data) == 0 {
		return 0
	}
	per := m.perRow()
	row := y
	if m.Height > 0 {
		row += min(m.top, max(m.rows()-m.Height, 0))
	}
	row = min(max(row, 0), m.rows()-1)
	col := 0
	switch hx := x - m.hexStart(); {
	case hx < 0:
		col = 0
	case x >= m.asciiStart():
		col = min(x-m.asciiStart(), per-1)
	case hx < m.hexWidth():
		for c := 0; c < per; c++ {
			if hx >= m.hexCol(c) {
				col = c
			}
		}
	default:
		col = per - 1
	}
	return min(row*per+col, len(m.Data)-1)
}

// updateMouse applies a mouse event when Mouse is on.
func (m Model) updateMouse(ev tui.MouseEvent) Model {
	if !m.Mouse {
		return m
	}
	switch {
	case ev.Button == tui.MouseButtonWheelUp || ev.Button == tui.MouseButtonWheelDown:
		if ev.Action != tui.MouseActionPress || !m.Bounds.Contains(ev.X, ev.Y) || m.Height <= 0 {
			return m
		}
		d := wheelRows
		if ev.Button == tui.MouseButtonWheelUp {
			d = -wheelRows
		}
		m.top = min(max(m.top+d, 0), max(m.rows()-m.Height, 0))
	case ev.Button != tui.MouseButtonLeft:
	case ev.Action == tui.MouseActionPress:
		if !m.Bounds.Contains(ev.X, ev.Y) {
			return m
		}
		m.cursor = m.byteAt(ev.X-m.Bounds.X, ev.Y-m.Bounds.Y)
		m.anchor = m.cursor
		m.dragging = true
		m.scrollToCursor()
	case ev.Action == tui.MouseActionMotion:
		if m.dragging {
			m.cursor = m.byteAt(ev.X-m.Bounds.X, ev.Y-m.Bounds.Y)
			m.scrollToCursor()
		}
	case ev.Action == tui.MouseActionRelease:
		if m.dragging && m.anchor == m.Cursor() {
			m.anchor = -1 // a click, not a drag
		}
		m.dragging = false
	}
	return m
}
