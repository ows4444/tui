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
	if m.LineLabel != nil {
		return m.labelWidth() + 1
	}
	if !m.ShowLineNumbers {
		return 0
	}
	return len(strconv.Itoa(m.value.lineCount())) + 1
}

// rowLines gives, for each row View shows, the 1-based line number it starts,
// or minus that number for a soft-wrapped continuation row.
func (m Model) rowLines() []int {
	if m.value.len() == 0 {
		return []int{1}
	}
	if m.wrapping() {
		rows, cur := m.allRows()
		top, bottom := 0, len(rows)
		if m.limit() > 0 {
			top = windowTop(m.top, cur, len(rows), m.limit())
			bottom = min(top+m.limit(), len(rows))
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
				out = append(out, -(line + 1))
			}
			if rows[i].last {
				line++
			}
		}
		return out
	}
	total := m.value.lineCount()
	top, bottom := 0, total
	if m.limit() > 0 {
		top = windowTop(m.top, m.value.lineOf(m.cursor), total, m.limit())
		bottom = min(top+m.limit(), total)
	}
	out := make([]int, 0, bottom-top)
	for i := top; i < bottom; i++ {
		out = append(out, i+1)
	}
	return out
}

// gutterText is row n's cell of the left column, g columns wide: the line
// number (or LineLabel's text) right-aligned in g-1 columns and a space, or
// blanks for a continuation row (n < 0) unless LineLabel names one.
func (m Model) gutterText(n, g int) string {
	var s string
	switch {
	case m.LineLabel != nil && n != 0:
		s = ansi.Truncate(m.LineLabel(max(n, -n), n < 0), g-1)
	case n > 0:
		s = strconv.Itoa(n)
	}
	return strings.Repeat(" ", max(g-1-ansi.Width(s), 0)) + s + " "
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
		lines[i] = st.Render(m.gutterText(n, g)) + l
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
		sub.SetString(0, y, m.gutterText(n, g), id)
	}
	return cellbuf.Rect{X: r.X + g, Y: r.Y, W: max(r.W-g, 0), H: r.H}
}

// limit is the number of rows View shows at most: Height, or, with MaxHeight
// and no Height, MaxHeight once the content has more rows than that. 0 means
// every row is shown.
func (m Model) limit() int {
	if m.Height > 0 || m.MaxHeight <= 0 {
		return m.Height
	}
	if m.value.lineCount() > m.MaxHeight {
		return m.MaxHeight
	}
	if m.wrapping() {
		if rows, _ := m.allRows(); len(rows) > m.MaxHeight {
			return m.MaxHeight
		}
	}
	return 0
}

// LineInfo says where the cursor is in the text as View lays it out.
type LineInfo struct {
	// Line is the 0-based line, as Line returns it.
	Line int
	// Column is the 0-based rune offset within the line, as Column returns it.
	Column int
	// Row is the 0-based visual row of the cursor within its line: always 0
	// without SoftWrap.
	Row int
	// Rows is how many visual rows the cursor's line takes: 1 without SoftWrap.
	Rows int
}

// LineInfo returns the cursor's line, column, and its visual row among the rows
// its line wraps onto, for a status bar ("Ln 3, Col 14") or a gutter.
func (m Model) LineInfo() LineInfo {
	line, col := m.lineCol()
	info := LineInfo{Line: line, Column: col, Rows: 1}
	if m.value.len() == 0 || !m.wrapping() {
		return info
	}
	rows, cur := m.allRows()
	first := cur
	for first > 0 && !rows[first-1].last {
		first--
	}
	last := cur
	for last < len(rows)-1 && !rows[last].last {
		last++
	}
	info.Row, info.Rows = cur-first, last-first+1
	return info
}

// labelWidth is the width of the LineLabel column without its trailing space.
func (m Model) labelWidth() int {
	return max(ansi.Width(m.LineLabel(1, false)), ansi.Width(m.LineLabel(m.value.lineCount(), false)))
}
