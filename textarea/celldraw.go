package textarea

import (
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/internal/edit"
)

// styleCache memoises the cell form of an ansi.Style, so drawing a frame does
// not parse the same SGR sequence again. It is bounded: past styleCacheMax
// entries a conversion is simply not stored.
var styleCache struct {
	sync.RWMutex
	m map[ansi.Style]cellbuf.Style
}

const styleCacheMax = 256

// cellStyle returns the cell style st renders as: the style a parse of
// st.Render("x") gives to its cell. A style the cell grid cannot represent is
// the default style.
func cellStyle(st ansi.Style) cellbuf.Style {
	styleCache.RLock()
	cs, ok := styleCache.m[st]
	styleCache.RUnlock()
	if ok {
		return cs
	}
	if b, err := cellbuf.Parse(st.Render("x")); err == nil {
		cs = b.Style(b.At(0, 0).Style)
	}
	styleCache.Lock()
	if styleCache.m == nil {
		styleCache.m = make(map[ansi.Style]cellbuf.Style)
	}
	if len(styleCache.m) < styleCacheMax {
		styleCache.m[st] = cs
	}
	styleCache.Unlock()
	return cs
}

// DrawCells draws the text area into the region r of buf, showing the same
// screen as View (the cursor cell, the placeholder, horizontal scroll, soft
// wrap and the Height window included) without building View's string. It
// makes Model a tui.CellDrawer: a Program whose root is, or composes through
// tui.DrawChild, a Model draws it straight into the frame's cell grid. Drawing
// is clipped to r.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	sub := buf.Sub(r)
	if sub.Width() == 0 || sub.Height() == 0 {
		return
	}
	text, cur := sub.StyleID(cellStyle(m.TextStyle)), sub.StyleID(cellStyle(m.CursorStyle))
	show := m.focused && m.cursorVisible

	if m.value.len() == 0 {
		ph := sub.StyleID(cellStyle(m.PlaceholderStyle))
		x0 := 0
		if show {
			sub.SetString(0, 0, " ", cur)
			x0 = 1
		}
		for y, rest, more := 0, m.Placeholder, true; more && y < sub.Height(); y++ {
			line := rest
			if i := strings.IndexByte(rest, '\n'); i >= 0 {
				line, rest = rest[:i], rest[i+1:]
			} else {
				more = false
			}
			sub.SetString(x0, y, line, ph)
			x0 = 0
		}
		return
	}

	if _, _, ok := m.selection(); ok {
		m.drawSelection(sub)
		return
	}

	if m.wrapping() {
		rows, curRow := m.allRows()
		top, bottom := 0, len(rows)
		if m.Height > 0 {
			top = windowTop(m.top, curRow, len(rows), m.Height)
			bottom = min(top+m.Height, len(rows))
		}
		for i := top; i < bottom && i-top < sub.Height(); i++ {
			row := rows[i]
			line := m.value.slice(row.s, row.e)
			cu := -1
			if show && i == curRow {
				cu = m.cursor - row.s
			}
			if drawASCII(sub, i-top, line, cu, row.last, text, cur) {
				continue
			}
			sp := span{cursor: -1}
			if show && i == curRow {
				sp = cursorSpan(string(line), nil, row.s, m.cursor, row.last)
			}
			drawRun(sub, i-top, string(line), sp, text, cur)
		}
		return
	}

	total := m.value.lineCount()
	curLine := m.value.lineOf(m.cursor)
	top, bottom := 0, total
	if m.Height > 0 {
		top = windowTop(m.top, curLine, total, m.Height)
		bottom = min(top+m.Height, total)
	}
	for i := top; i < bottom && i-top < sub.Height(); i++ {
		ls := m.value.lineStart(i)
		lr := m.value.slice(ls, m.value.lineEnd(i))
		lineCursor := -1
		if i == curLine {
			lineCursor = m.cursor - ls
		}
		if m.Width == 0 {
			cu := -1
			if show {
				cu = lineCursor
			}
			if drawASCII(sub, i-top, lr, cu, true, text, cur) {
				continue
			}
			sp := span{cursor: -1}
			s := string(lr)
			if show && lineCursor >= 0 {
				sp = cursorSpan(s, nil, 0, lineCursor, true)
			}
			drawRun(sub, i-top, s, sp, text, cur)
			continue
		}
		m.drawWindowed(sub, i-top, lr, lineCursor, show, text, cur)
	}
}

// drawWindowed draws line lr scrolled horizontally to Width columns, as
// renderLine does; lineCursor is the cursor's rune offset or -1.
func (m Model) drawWindowed(sub *cellbuf.Buffer, y int, lr []rune, lineCursor int, show bool, text, cur cellbuf.StyleID) {
	var e edit.Editor
	e.Set(string(lr), 0)
	if lineCursor >= 0 {
		e.SetCursor(len(edit.Split(string(lr[:lineCursor]))))
	} else {
		e.SetCursor(0)
	}
	start, end := e.Window(m.Width)
	cl := make([]string, 0, end-start)
	base := 0
	for i := 0; i < end; i++ {
		c := e.Cluster(i)
		if i < start {
			base += utf8.RuneCountInString(c)
		} else {
			cl = append(cl, c)
		}
	}
	sp := span{cursor: -1}
	if show && lineCursor >= 0 {
		sp = cursorSpan("", cl, base, lineCursor, end == e.Len() && lineCursor == len(lr))
	}
	drawRun(sub, y, strings.Join(cl, ""), sp, text, cur)
}

// span locates the cursor cell within a row's text: bytes [from, to) of it, or
// the cell just past the end when end is set. cursor < 0 means no cursor.
type span struct {
	from, to int
	end      bool
	cursor   int
}

// cursorSpan finds the cluster that starts at rune position cursor in a row
// whose first rune is at position base. The row is cl when non-nil, else the
// clusters of s. atEnd allows the cursor to sit after the last cluster.
func cursorSpan(s string, cl []string, base, cursor int, atEnd bool) span {
	none := span{cursor: -1}
	if cl == nil && isASCII(s) {
		if i := cursor - base; i >= 0 && i < len(s) {
			return span{from: i, to: i + 1}
		} else if i == len(s) && atEnd {
			return span{from: i, to: i, end: true}
		}
		return none
	}
	if cl == nil {
		cl = edit.Split(s)
	}
	pos, b := base, 0
	for _, c := range cl {
		if pos == cursor {
			return span{from: b, to: b + len(c)}
		}
		pos += utf8.RuneCountInString(c)
		b += len(c)
	}
	if pos == cursor && atEnd {
		return span{from: b, to: b, end: true}
	}
	return none
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] >= 0x7f {
			return false
		}
	}
	return true
}

// drawASCII draws line lr on row y as drawRun would, without building a string,
// when every rune of it is printable ASCII (one cell each); it reports false,
// having drawn nothing, otherwise. cursor is the rune index of the cursor cell,
// or -1; atEnd lets the cursor sit just past the last rune.
func drawASCII(sub *cellbuf.Buffer, y int, lr []rune, cursor int, atEnd bool, textID, curID cellbuf.StyleID) bool {
	for _, c := range lr {
		if c < 0x20 || c >= 0x7f {
			return false
		}
	}
	if cursor < 0 || cursor > len(lr) || cursor == len(lr) && !atEnd {
		sub.SetRunes(0, y, lr, textID)
		return true
	}
	x := sub.SetRunes(0, y, lr[:cursor], textID)
	if cursor == len(lr) {
		sub.SetString(x, y, " ", curID)
		return true
	}
	x += sub.SetRunes(x, y, lr[cursor:cursor+1], curID)
	sub.SetRunes(x, y, lr[cursor+1:], textID)
	return true
}

// drawRun writes text on row y in the text style, with the cursor cell of sp
// in the cursor style.
func drawRun(sub *cellbuf.Buffer, y int, text string, sp span, textID, curID cellbuf.StyleID) {
	if sp.cursor < 0 {
		sub.SetString(0, y, text, textID)
		return
	}
	x := 0
	if sp.from > 0 {
		x += sub.SetString(0, y, text[:sp.from], textID)
	}
	if sp.end {
		sub.SetString(x, y, " ", curID)
		return
	}
	x += sub.SetString(x, y, text[sp.from:sp.to], curID)
	if sp.to < len(text) {
		sub.SetString(x, y, text[sp.to:], textID)
	}
}
