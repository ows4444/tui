package textarea

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

// Line returns the 0-based index of the line the cursor is on. Lines are
// split on '\n'; soft wrap does not add lines.
func (m Model) Line() int {
	l, _ := m.lineCol()
	return l
}

// Column returns the cursor's 0-based offset, in runes, within its line.
func (m Model) Column() int {
	_, c := m.lineCol()
	return c
}

// LineCount returns the number of lines in the value: 1 for an empty value.
func (m Model) LineCount() int { return m.value.lineCount() }

// SetCursorPosition moves the cursor to line and column (a rune offset in the
// line), both 0-based and clamped: a line past the last is the last, a column
// past the end of its line is the line's end. The selection is cleared.
func (m *Model) SetCursorPosition(line, col int) {
	line = clamp(line, 0, m.value.lineCount()-1)
	ls, le := m.value.lineStart(line), m.value.lineEnd(line)
	m.cursor = m.snap(clamp(ls+max(col, 0), ls, le))
	m.sel = 0
	m.scrollToCursor()
}

// InsertString inserts s at the cursor, in place of the selection if there is
// one, as one undo step, and leaves the cursor after it. Text beyond CharLimit
// is dropped, as for typing. "\r\n" and "\r" become "\n".
func (m *Model) InsertString(s string) {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	m.insertRunes([]rune(s), false)
	m.scrollToCursor()
}

// gutter is the width of the line-number column: the digits of the highest
// line number and a space, or 0 when ShowLineNumbers is off.
func (m Model) gutter() int {
	if !m.ShowLineNumbers {
		return 0
	}
	return len(strconv.Itoa(m.value.lineCount())) + 1
}

// rowLines gives, for each row View shows, the 1-based line number it starts,
// or 0 for a soft-wrapped continuation row.
func (m Model) rowLines() []int {
	if m.value.len() == 0 {
		return []int{1}
	}
	if m.wrapping() {
		rows, cur := m.allRows()
		top, bottom := 0, len(rows)
		if m.Height > 0 {
			top = windowTop(m.top, cur, len(rows), m.Height)
			bottom = min(top+m.Height, len(rows))
		}
		line := 0
		for i := 0; i < top; i++ {
			if rows[i].last {
				line++
			}
		}
		out := make([]int, 0, bottom-top)
		for i := top; i < bottom; i++ {
			first := i == 0 || rows[i-1].last
			if first {
				out = append(out, line+1)
			} else {
				out = append(out, 0)
			}
			if rows[i].last {
				line++
			}
		}
		return out
	}
	total := m.value.lineCount()
	top, bottom := 0, total
	if m.Height > 0 {
		top = windowTop(m.top, m.value.lineOf(m.cursor), total, m.Height)
		bottom = min(top+m.Height, total)
	}
	out := make([]int, 0, bottom-top)
	for i := top; i < bottom; i++ {
		out = append(out, i+1)
	}
	return out
}

// gutterText is row n's line-number cell: the number right-aligned in g-1
// columns and a space, or blanks for a continuation row (n == 0).
func gutterText(n, g int) string {
	if n == 0 {
		return strings.Repeat(" ", g)
	}
	s := strconv.Itoa(n)
	return strings.Repeat(" ", max(g-1-len(s), 0)) + s + " "
}

// withGutter prefixes each line of body with its line number.
func (m Model) withGutter(body string) string {
	g := m.gutter()
	nums := m.rowLines()
	st := m.lineNumberStyle()
	lines := strings.Split(body, "\n")
	for i, l := range lines {
		n := 0
		if i < len(nums) {
			n = nums[i]
		}
		lines[i] = st.Render(gutterText(n, g)) + l
	}
	return strings.Join(lines, "\n")
}

// lineNumberStyle is LineNumberStyle, or faint text when it is the zero value.
func (m Model) lineNumberStyle() ansi.Style {
	if m.LineNumberStyle == (ansi.Style{}) {
		return ansi.NewStyle().Faint()
	}
	return m.LineNumberStyle
}

// drawGutter draws the line-number column into the left g cells of r and
// returns the rectangle that is left for the text.
func (m Model) drawGutter(buf *cellbuf.Buffer, r cellbuf.Rect) cellbuf.Rect {
	g := m.gutter()
	sub := buf.Sub(r)
	id := sub.StyleID(cellStyle(m.lineNumberStyle()))
	for y, n := range m.rowLines() {
		if y >= sub.Height() {
			break
		}
		sub.SetString(0, y, gutterText(n, g), id)
	}
	return cellbuf.Rect{X: r.X + g, Y: r.Y, W: max(r.W-g, 0), H: r.H}
}
