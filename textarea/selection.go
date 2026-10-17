package textarea

import (
	"strings"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/internal/edit"
)

// SelectedText returns the selected text, or "" when nothing is selected.
func (m Model) SelectedText() string {
	lo, hi, ok := m.selection()
	if !ok {
		return ""
	}
	return m.value.str(lo, hi)
}

// Selection returns the selected rune range [lo, hi) of Value; ok is false when
// nothing is selected.
func (m Model) Selection() (lo, hi int, ok bool) { return m.selection() }

// SelectAll selects the whole value and moves the cursor to its end.
func (m *Model) SelectAll() {
	if m.value.len() == 0 {
		return
	}
	m.sel, m.cursor = 1, m.value.len()
	m.scrollToCursor()
}

func (m Model) selStyle() ansi.Style {
	if m.SelectionStyle == (ansi.Style{}) {
		return ansi.NewStyle().Reverse()
	}
	return m.SelectionStyle
}

// srow is one visible row as clusters: base is the rune offset of cl[0], end
// says the cursor may sit just past the last cluster, own that the cursor is on
// this row.
type srow struct {
	cl   []string
	base int
	end  bool
	own  bool
}

// visibleRows returns the rows View shows (soft wrap, horizontal window and
// Height window applied), for the paths that style per cluster.
func (m Model) visibleRows() []srow {
	if m.wrapping() {
		rows, cur := m.allRows()
		top, bottom := 0, len(rows)
		if m.limit() > 0 {
			top = windowTop(m.top, cur, len(rows), m.limit())
			bottom = min(top+m.limit(), len(rows))
		}
		out := make([]srow, 0, bottom-top)
		for i := top; i < bottom; i++ {
			r := rows[i]
			out = append(out, srow{cl: edit.Split(m.value.str(r.s, r.e)), base: r.s, end: r.last, own: i == cur})
		}
		return out
	}
	total := m.value.lineCount()
	curLine := m.value.lineOf(m.cursor)
	top, bottom := 0, total
	if m.limit() > 0 {
		top = windowTop(m.top, curLine, total, m.limit())
		bottom = min(top+m.limit(), total)
	}
	out := make([]srow, 0, bottom-top)
	for i := top; i < bottom; i++ {
		ls := m.value.lineStart(i)
		text := m.value.str(ls, m.value.lineEnd(i))
		row := srow{base: ls, end: true, own: i == curLine}
		if m.Width <= 0 {
			row.cl = edit.Split(text)
		} else {
			var e edit.Editor
			e.Set(text, 0)
			if i == curLine {
				e.SetCursor(len(edit.Split(m.value.str(ls, m.cursor))))
			} else {
				e.SetCursor(0)
			}
			start, end := e.Window(m.Width)
			for j := 0; j < end; j++ {
				if j < start {
					row.base += utf8.RuneCountInString(e.Cluster(j))
				} else {
					row.cl = append(row.cl, e.Cluster(j))
				}
			}
			row.end = end == e.Len()
		}
		out = append(out, row)
	}
	return out
}

// cellKind says how a cluster is drawn.
type cellKind int

const (
	kindText cellKind = iota
	kindSel
	kindCursor
)

// walk calls f for every cell run of row r in drawing order: each cluster with
// its kind, then the end-of-line cursor cell (text " ") when it is shown.
func (m Model) walk(r srow, f func(text string, k cellKind, pos int)) {
	lo, hi, _ := m.selection()
	show := r.own && m.focused && m.cursorVisible
	pos := r.base
	for _, c := range r.cl {
		switch {
		case show && pos == m.cursor:
			f(c, kindCursor, pos)
		case pos >= lo && pos < hi:
			f(c, kindSel, pos)
		default:
			f(c, kindText, pos)
		}
		pos += utf8.RuneCountInString(c)
	}
	if show && r.end && pos == m.cursor {
		f(" ", kindCursor, pos)
	}
}

// rowStyles returns the Highlighter's style for each rune of the line r is
// on, and the rune offset where that line starts. It is nil, 0 without a
// Highlighter or when its answer does not add up to the line.
func (m Model) rowStyles(r srow) ([]ansi.Style, int) {
	if m.Highlighter == nil {
		return nil, 0
	}
	line := m.value.lineOf(r.base)
	ls := m.value.lineStart(line)
	text := m.value.str(ls, m.value.lineEnd(line))
	sty := make([]ansi.Style, 0, len(text))
	for _, sp := range m.Highlighter(text).WithDefault(m.TextStyle) {
		for range sp.Text {
			sty = append(sty, sp.Style)
		}
	}
	if len(sty) != utf8.RuneCountInString(text) {
		return nil, 0
	}
	return sty, ls
}

// textStyle is the style of the text cluster at rune offset pos: the
// Highlighter's, or TextStyle.
func (m Model) textStyle(sty []ansi.Style, ls, pos int) ansi.Style {
	if i := pos - ls; i >= 0 && i < len(sty) {
		return sty[i]
	}
	return m.TextStyle
}

// styledRow calls f for every run of row r in drawing order with the style to
// draw it in: the cursor, the selection, or the text in its Highlighter or
// TextStyle colour with the editor aids over it. a is the frame's aids.
func (m Model) styledRow(r srow, a aids, f func(text string, st ansi.Style)) {
	sty, ls := m.rowStyles(r)
	sel := m.selStyle()
	a = a.forRow(r)
	x := 0
	m.walk(r, func(c string, k cellKind, pos int) {
		st, text := m.CursorStyle, c
		switch k {
		case kindSel:
			st = sel
		case kindText:
			st, text = a.apply(m.textStyle(sty, ls, pos), c, pos, x)
		}
		f(text, st)
		x += ansi.Width(c)
	})
	if a.cursorLine || len(m.Ruler) > 0 {
		a.tail(x, m.TextStyle, f)
	}
}

// selectionView is View when text is selected, or an aid or a Highlighter is
// set.
func (m Model) selectionView() string {
	rows := m.visibleRows()
	out := make([]string, len(rows))
	a := m.newAids()
	for i, r := range rows {
		var b strings.Builder
		m.styledRow(r, a, func(text string, st ansi.Style) { b.WriteString(st.Render(text)) })
		out[i] = b.String()
	}
	return strings.Join(out, "\n")
}

// drawSelection is DrawCells when text is selected, or an aid or a Highlighter
// is set.
func (m Model) drawSelection(sub *cellbuf.Buffer) {
	a := m.newAids()
	for y, r := range m.visibleRows() {
		if y >= sub.Height() {
			break
		}
		x := 0
		m.styledRow(r, a, func(text string, st ansi.Style) {
			x += sub.SetString(x, y, text, sub.StyleID(cellStyle(st)))
		})
	}
}
