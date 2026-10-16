package viewport

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

// point is a cell of the content: a line (an index into Model.lines, which
// includes the dropped head) and a display column within that line.
type point struct{ line, col int }

func (p point) before(q point) bool { return p.line < q.line || (p.line == q.line && p.col < q.col) }

// selection is the pointer's selected text: from anchor to head, both cells
// included, in either order.
type selection struct {
	anchor, head point
	active       bool // there is a selection (anchor and head differ)
	dragging     bool // the left button is down
}

// bounds returns the ends in reading order.
func (s selection) bounds() (a, b point) {
	if s.head.before(s.anchor) {
		return s.head, s.anchor
	}
	return s.anchor, s.head
}

// shift moves both ends up by n lines (a compaction of the line store); an end
// that falls off the front ends the selection.
func (s selection) shift(n int) selection {
	s.anchor.line -= n
	s.head.line -= n
	if s.anchor.line < 0 || s.head.line < 0 {
		return selection{}
	}
	return s
}

// afterTrim ends the selection if TrimFront dropped a line it touches.
func (s selection) afterTrim(head int) selection {
	if s.active && (s.anchor.line < head || s.head.line < head) {
		return selection{}
	}
	return s
}

// SelectedText returns the text under the selection as plain text, lines joined
// with "\n", or "" when nothing is selected. It is what Copy puts on the
// clipboard.
func (m Model) SelectedText() string {
	if !m.sel.active {
		return ""
	}
	a, b := m.sel.bounds()
	var out []string
	for l := a.line; l <= b.line && l < len(m.lines); l++ {
		if l < m.head {
			continue
		}
		plain := ansi.StripANSI(m.lines[l])
		from, to := 0, ansi.Width(plain)
		if l == a.line {
			from = a.col
		}
		if l == b.line {
			to = min(to, b.col+1)
		}
		out = append(out, sliceCols(plain, from, to))
	}
	return strings.Join(out, "\n")
}

// ClearSelection removes the selection.
func (m *Model) ClearSelection() { m.sel = selection{} }

// SelectAll selects all the content.
func (m *Model) SelectAll() {
	if m.LineCount() == 0 {
		return
	}
	last := len(m.lines) - 1
	m.sel = selection{
		anchor: point{line: m.head},
		head:   point{line: last, col: max(ansi.Width(ansi.StripANSI(m.lines[last]))-1, 0)},
		active: true,
	}
}

// sliceCols returns the display columns [from, to) of plain text.
func sliceCols(plain string, from, to int) string {
	if to <= from {
		return ""
	}
	return ansi.Truncate(ansi.TrimLeftWidth(plain, from), to-from)
}

// cellAt maps a screen cell to a content cell. y is clamped to the rows shown,
// and a column past the end of a line is its last cell. ok is false when the
// viewport has no content.
func (m Model) cellAt(x, y int) (point, bool) {
	if m.total() == 0 {
		return point{}, false
	}
	row := min(max(y-m.Bounds.Y, 0), m.Height-1) + m.yOffset
	row = min(row, m.total()-1)
	col := max(x-m.Bounds.X, 0)
	if m.wrapOn() {
		r := m.vrows()[row]
		return point{line: r.line, col: r.col + min(col, max(ansi.Width(r.text)-1, 0))}, true
	}
	line := m.head + row
	return point{line: line, col: min(col+m.xOffset, max(ansi.Width(ansi.StripANSI(m.lines[line]))-1, 0))}, true
}

// updateSelection applies a left-button press, drag and release when Mouse is
// on: a press inside Bounds starts a selection, motion with the button down
// extends it (scrolling a line when the pointer is past the top or bottom row),
// and a release keeps it, or ends it if nothing was dragged.
func (m Model) updateSelection(ev tui.MouseEvent) Model {
	if !m.Mouse || ev.Button != tui.MouseButtonLeft {
		return m
	}
	switch ev.Action {
	case tui.MouseActionPress:
		if !m.Bounds.Contains(ev.X, ev.Y) {
			return m
		}
		p, ok := m.cellAt(ev.X, ev.Y)
		if !ok {
			return m
		}
		m.sel = selection{anchor: p, head: p, dragging: true}
	case tui.MouseActionMotion:
		if !m.sel.dragging {
			return m
		}
		switch {
		case ev.Y < m.Bounds.Y:
			m.LineUp(1)
		case ev.Y >= m.Bounds.Y+m.Height:
			m.LineDown(1)
		}
		if p, ok := m.cellAt(ev.X, ev.Y); ok {
			m.sel.head = p
			m.sel.active = m.sel.anchor != p
		}
	case tui.MouseActionRelease:
		m.sel.dragging = false
		if m.sel.anchor == m.sel.head {
			m.sel = selection{}
		}
	}
	return m
}

// paint draws content row i with the selection over any search highlight.
func (m Model) paint(vr []vrow, i int) string {
	text := m.paintSearch(vr, i)
	if !m.sel.active {
		return text
	}
	var idx, c0 int
	if m.wrapOn() {
		idx, c0 = vr[i].line, vr[i].col
	} else {
		idx = m.head + i
	}
	a, b := m.sel.bounds()
	if idx < a.line || idx > b.line {
		return text
	}
	w := ansi.Width(ansi.StripANSI(text))
	from, to := 0, c0+w
	if idx == a.line {
		from = a.col
	}
	if idx == b.line {
		to = min(to, b.col+1)
	}
	from, to = max(from-c0, 0), min(to-c0, w)
	if to <= from {
		return text
	}
	return restyle(text, from, to, m.selectionStyle())
}

func (m Model) selectionStyle() ansi.Style {
	if m.SelectionStyle == (ansi.Style{}) {
		return ansi.NewStyle().Reverse()
	}
	return m.SelectionStyle
}

// restyle adds sel's look to display columns [from, to) of the styled text, and
// leaves the text as it is if the cell grid cannot represent it.
func restyle(text string, from, to int, sel ansi.Style) string {
	buf, err := cellbuf.Parse(text)
	if err != nil {
		return text
	}
	probe, err := cellbuf.Parse(sel.Render("x"))
	if err != nil {
		return text
	}
	over := probe.Style(probe.At(0, 0).Style)
	for x := from; x < to && x < buf.Width(); x++ {
		c := buf.At(x, 0)
		if c.Width == 0 {
			continue
		}
		st := buf.Style(c.Style)
		st.Attrs |= over.Attrs
		if over.FG != 0 {
			st.FG = over.FG
		}
		if over.BG != 0 {
			st.BG = over.BG
		}
		buf.SetString(x, 0, c.Cluster, buf.StyleID(st))
	}
	return buf.String()
}
