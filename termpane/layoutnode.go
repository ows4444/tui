package termpane

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/vt"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the pane to a layout.Node of the emulator's size, which
// crops or pads to the room it is given. Call SetSize to make the terminal the
// size of the room. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return paneNode{m} }

type paneNode struct{ m Model }

func (n paneNode) Measure(c layout.Constraints) layout.Size {
	cols, rows := n.m.Size()
	return c.Constrain(layout.Size{W: cols, H: rows})
}

func (n paneNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	return strings.Join(n.m.rows(s), "\n")
}

// View renders the pane at the terminal's own size.
func (m Model) View() string {
	cols, rows := m.Size()
	return m.LayoutNode().Render(layout.Size{W: cols, H: rows})
}

// rows draws the screen as s.H lines of s.W columns.
func (m Model) rows(s layout.Size) []string {
	out := make([]string, s.H)
	if m.s == nil {
		for i := range out {
			out[i] = strings.Repeat(" ", s.W)
		}
		return out
	}
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	t := m.s.term
	if t == nil {
		for i := range out {
			out[i] = strings.Repeat(" ", s.W)
		}
		return out
	}
	cx, cy, visible := t.Cursor()
	showCursor := m.focused && visible && m.offset == 0
	for y := range out {
		cells := t.Row(m.offset, y)
		cursorX := -1
		if showCursor && y == cy {
			cursorX = cx
		}
		out[y] = drawRow(cells, s.W, cursorX)
	}
	if m.offset > 0 {
		out[0] = m.badge(out[0], s.W)
	}
	return out
}

// badge writes how far back the view is at the right end of the first row.
func (m Model) badge(row string, w int) string {
	text := " scrollback " + strconv.Itoa(m.offset) + " "
	if ansi.Width(text) >= w {
		return row
	}
	st := ansi.NewStyle().Reverse().Bold()
	keep := w - ansi.Width(text)
	return ansi.Truncate(row, keep) + strings.Repeat(" ", max(keep-ansi.Width(ansi.Truncate(row, keep)), 0)) + st.Render(text)
}

// drawRow is one row of cells as a styled string of exactly w columns; the cell
// at cursorX (or none when it is negative) is drawn reversed.
func drawRow(cells []vt.Cell, w, cursorX int) string {
	var b strings.Builder
	var run strings.Builder
	var runStyle ansi.Style
	have := false
	flush := func() {
		if have {
			b.WriteString(runStyle.Render(run.String()))
		}
		run.Reset()
		have = false
	}
	cols := 0
	for x := 0; x < len(cells) && cols < w; x++ {
		c := cells[x]
		if c.Width == 0 {
			continue
		}
		if cols+int(c.Width) > w { // a wide cluster cut by the pane's edge
			break
		}
		text := c.Cluster
		if c.Attrs&vt.Conceal != 0 {
			text = strings.Repeat(" ", int(c.Width))
		}
		st := cellStyle(c, x == cursorX)
		if !have || st != runStyle {
			flush()
			runStyle, have = st, true
		}
		run.WriteString(ansi.Sanitize(text))
		cols += int(c.Width)
	}
	flush()
	if cols < w {
		b.WriteString(strings.Repeat(" ", w-cols))
	}
	return b.String()
}

// cellStyle is the look of a cell.
func cellStyle(c vt.Cell, cursor bool) ansi.Style {
	st := ansi.NewStyle()
	if c.FG != nil {
		st = st.Foreground(c.FG)
	}
	if c.BG != nil {
		st = st.Background(c.BG)
	}
	a := c.Attrs
	if cursor {
		a ^= vt.Reverse
	}
	if a&vt.Bold != 0 {
		st = st.Bold()
	}
	if a&vt.Dim != 0 {
		st = st.Faint()
	}
	if a&vt.Italic != 0 {
		st = st.Italic()
	}
	if a&vt.Underline != 0 {
		st = st.Underline()
	}
	if a&vt.Blink != 0 {
		st = st.Blink()
	}
	if a&vt.Reverse != 0 {
		st = st.Reverse()
	}
	if a&vt.Strike != 0 {
		st = st.Strikethrough()
	}
	return st
}

// Linearize describes the pane as plain text for accessible output (see
// tui.Linearizer): the title when the program set one, then the visible rows
// with trailing blanks trimmed.
func (m Model) Linearize() string {
	cols, rows := m.Size()
	var b strings.Builder
	b.WriteString("Terminal")
	if t := m.Title(); t != "" {
		b.WriteString(" \"" + ansi.Sanitize(t) + "\"")
	}
	if m.s == nil {
		return b.String()
	}
	m.s.mu.Lock()
	defer m.s.mu.Unlock()
	if m.s.term == nil {
		return b.String()
	}
	for y := 0; y < rows; y++ {
		var line strings.Builder
		for x, c := range m.s.term.Row(0, y) {
			if x < cols && c.Width > 0 {
				line.WriteString(ansi.Sanitize(c.Cluster))
			}
		}
		b.WriteString("\n" + strings.TrimRight(line.String(), " "))
	}
	return strings.TrimRight(b.String(), "\n ")
}
