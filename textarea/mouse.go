package textarea

import (
	"unicode/utf8"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/edit"
)

// wheelRows is how many rows one wheel notch moves the cursor.
const wheelRows = 3

// firstRow returns the index of the first visible row: a visual row under soft
// wrap, a line otherwise.
func (m Model) firstRow() int {
	if m.limit() <= 0 {
		return 0
	}
	if m.wrapping() {
		rows, cur := m.allRows()
		return windowTop(m.top, cur, len(rows), m.limit())
	}
	return windowTop(m.top, m.value.lineOf(m.cursor), m.value.lineCount(), m.limit())
}

// offsetAt returns the rune offset of the cluster boundary under the cell at
// column x, row y of the area View draws (0,0 is its top-left cell). It
// accounts for the Height window, soft wrap, a line scrolled horizontally and
// wide clusters; a cell past the end of a line is its end, one beyond the last
// row the end of the text.
func (m Model) offsetAt(x, y int) int {
	if m.value.len() == 0 {
		return 0
	}
	x = max(x-m.gutter(), 0) // a click in the line numbers is the line's start
	top := m.firstRow()
	if m.wrapping() {
		rows, _ := m.allRows()
		return m.colInRow(rows[clamp(top+y, 0, len(rows)-1)], x)
	}
	line := clamp(top+y, 0, m.value.lineCount()-1)
	ls, le := m.value.lineStart(line), m.value.lineEnd(line)
	if m.Width <= 0 {
		return ls + offsetForCol(m.value.slice(ls, le), x)
	}
	// Horizontally scrolled: find the window the way View does.
	var e edit.Editor
	e.Set(m.value.str(ls, le), 0)
	if line == m.value.lineOf(m.cursor) {
		e.SetCursor(len(edit.Split(m.value.str(ls, m.cursor))))
	}
	start, end := e.Window(m.Width)
	pos, acc := ls, 0
	for i := 0; i < end; i++ {
		c := e.Cluster(i)
		if i >= start {
			acc += ansi.Width(c)
			if x < acc {
				return pos
			}
		}
		pos += utf8.RuneCountInString(c)
	}
	if end < e.Len() {
		return pos - utf8.RuneCountInString(e.Cluster(end-1))
	}
	return le
}

// updateMouse applies a mouse event when Mouse is on: a left press inside
// Bounds puts the cursor on the clicked cell and starts a selection there, a
// drag extends it, a release ends it, and the wheel moves the cursor.
func (m Model) updateMouse(ev tui.MouseEvent) Model {
	if !m.Mouse {
		return m
	}
	switch {
	case ev.Button == tui.MouseButtonWheelUp || ev.Button == tui.MouseButtonWheelDown:
		if ev.Action != tui.MouseActionPress || !m.Bounds.Contains(ev.X, ev.Y) {
			return m
		}
		d := wheelRows
		if ev.Button == tui.MouseButtonWheelUp {
			d = -wheelRows
		}
		m.sel = 0
		for i := 0; i < wheelRows; i++ {
			m.moveVertical(d / wheelRows)
		}
		m.scrollToCursor()
	case ev.Button != tui.MouseButtonLeft:
	case ev.Action == tui.MouseActionPress:
		if !m.Bounds.Contains(ev.X, ev.Y) {
			return m
		}
		m.cursor = m.offsetAt(ev.X-m.Bounds.X, ev.Y-m.Bounds.Y)
		m.sel = m.cursor + 1
		m.dragging = true
		switch {
		case ev.Clicks >= 3: // the whole line
			start, end := m.currentLineBounds()
			m.sel, m.cursor, m.dragging = start+1, end, false
		case ev.Clicks == 2: // the word
			if lo, hi, ok := m.runAt(m.cursor); ok {
				m.sel, m.cursor, m.dragging = lo+1, hi, false
			}
		}
		m.cursorVisible = true
		m.scrollToCursor()
	case ev.Action == tui.MouseActionMotion:
		if m.dragging {
			m.cursor = m.offsetAt(ev.X-m.Bounds.X, ev.Y-m.Bounds.Y)
			m.cursorVisible = true
			m.scrollToCursor()
		}
	case ev.Action == tui.MouseActionRelease:
		m.dragging = false
	}
	return m
}

// runAt is the run of spaces or of non-spaces around rune offset pos within its
// line, for a double click; ok is false on an empty line.
func (m Model) runAt(pos int) (lo, hi int, ok bool) {
	start, end := m.boundsAt(pos)
	if end <= start {
		return 0, 0, false
	}
	at := min(max(pos, start), end-1)
	blank := func(i int) bool { r := m.value.at(i); return r == ' ' || r == '\t' }
	class := blank(at)
	lo, hi = at, at+1
	for lo > start && blank(lo-1) == class {
		lo--
	}
	for hi < end && blank(hi) == class {
		hi++
	}
	return lo, hi, true
}
