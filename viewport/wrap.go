package viewport

import (
	"sort"

	"github.com/ows4444/tui/ansi"
)

// vrow is one visual row of the soft-wrapped content: a slice of logical line
// `line` (an index into Model.lines, not the live slice) that starts col
// display columns into that line. text is self-contained styled text.
type vrow struct {
	line int
	col  int
	text string
}

// wrapOn reports whether soft wrap is in effect.
func (m Model) wrapOn() bool { return m.SoftWrap && m.Width > 0 }

// wrapLine splits line into chunks of at most w display columns, never
// splitting a grapheme cluster. Each chunk carries the SGR state active where
// it starts, so it can be drawn on its own.
func wrapLine(line string, w int) []string {
	if pw, ok := ansi.PlainASCIIWidth(line); ok {
		if pw <= w {
			return []string{line}
		}
		out := make([]string, 0, pw/w+1)
		for len(line) > w {
			out = append(out, line[:w])
			line = line[w:]
		}
		return append(out, line)
	}
	var out []string
	rest := line
	for ansi.Width(rest) > w {
		chunk := ansi.Truncate(rest, w)
		cw := ansi.Width(chunk)
		if cw <= 0 { // a cluster wider than w: cannot split further
			break
		}
		out = append(out, chunk)
		rest = ansi.TrimLeftWidth(rest, cw)
	}
	return append(out, rest)
}

// buildRows appends the visual rows of m.lines[from:to] to rows.
func (m Model) buildRows(rows []vrow, from, to int) []vrow {
	for i := from; i < to; i++ {
		col := 0
		for _, c := range wrapLine(m.lines[i], m.Width) {
			rows = append(rows, vrow{line: i, col: col, text: c})
			col += ansi.Width(c)
		}
	}
	return rows
}

// sync brings the visual-row cache up to date after a mutation. full forces a
// rebuild (content replaced or renumbered).
func (m *Model) sync(full bool) {
	if !m.wrapOn() {
		m.wrows, m.wwidth, m.wcovered = nil, 0, 0
		return
	}
	if full || m.wwidth != m.Width || m.wcovered > len(m.lines) || m.wfrom > m.head {
		m.wrows = m.buildRows(nil, m.head, len(m.lines))
		m.wwidth, m.wfrom = m.Width, m.head
	} else if m.wcovered < len(m.lines) {
		m.wrows = m.buildRows(m.wrows, m.wcovered, len(m.lines))
	}
	m.wcovered = len(m.lines)
}

// vrows returns the visual rows of the live content. It uses the cache when
// current and otherwise builds them afresh (O(content)); Update and the
// mutating methods keep the cache current.
func (m Model) vrows() []vrow {
	if m.wwidth == m.Width && m.wcovered == len(m.lines) && m.wfrom <= m.head {
		off := sort.Search(len(m.wrows), func(i int) bool { return m.wrows[i].line >= m.head })
		return m.wrows[off:]
	}
	return m.buildRows(nil, m.head, len(m.lines))
}

// total is the number of scrollable rows: visual rows when wrapping, else lines.
func (m Model) total() int {
	if m.wrapOn() {
		return len(m.vrows())
	}
	return m.LineCount()
}
