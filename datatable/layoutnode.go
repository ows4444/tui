package datatable

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/widgets"
)

// LayoutNode adapts the table to a layout.Node. Measure reports its natural
// size: the columns at their content widths (two spaces apart) by one row per
// record plus the header and divider. Render fits the table to the allotted
// Size: when too narrow it shrinks the widest columns first and truncates
// cells with an ellipsis, and when too short it pins the header and shows a
// window of rows that keeps the cursor row visible. The cursor row keeps its
// highlight; the Model is not changed.
func (m Model) LayoutNode() layout.Node { return tableNode{m} }

type tableNode struct{ m Model }

// gap is the spacing widgets.TableRows puts between columns.
const gap = 2

// naturalWidths returns the width of each column: ColumnWidths[i] when that
// is positive, otherwise the widest header or cell, served from the cache.
func (n tableNode) naturalWidths() []int {
	m := n.m
	fixed := func(i int) int {
		if i < len(m.ColumnWidths) {
			return m.ColumnWidths[i]
		}
		return 0
	}
	w := make([]int, len(m.Headers))
	var nat []int
	for i := range w {
		if f := fixed(i); f > 0 {
			w[i] = f
			continue
		}
		if nat == nil {
			nat = m.measure()
		}
		w[i] = nat[i]
	}
	return w
}

func sum(v []int) int {
	t := 0
	for _, x := range v {
		t += x
	}
	return t
}

func (n tableNode) Measure(c layout.Constraints) layout.Size {
	if len(n.m.Headers) == 0 {
		return c.Constrain(layout.Size{})
	}
	w := n.naturalWidths()
	return c.Constrain(layout.Size{W: cursorGutter + sum(w) + gap*(len(w)-1), H: len(n.m.Rows) + 2})
}

// fitWidths shrinks the widest column one cell at a time until the columns
// plus their gaps fit total, never below one cell each.
func fitWidths(w []int, total int) []int {
	out := append([]int(nil), w...)
	over := sum(out) + gap*(len(out)-1) - total
	for ; over > 0; over-- {
		widest := 0
		for i := range out {
			if out[i] > out[widest] {
				widest = i
			}
		}
		if out[widest] <= 1 {
			break
		}
		out[widest]--
	}
	return out
}

// clip fits cell into width cells, ending in an ellipsis when it is cut.
func clip(cell string, width int, ellipsis string) string {
	if ansi.Width(cell) <= width {
		return cell
	}
	if width <= 1 {
		return ansi.Truncate(cell, width)
	}
	return ansi.Truncate(cell, width-1) + ellipsis
}

func (n tableNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	if len(m.Headers) == 0 {
		return layout.Block("").Render(s)
	}
	ellipsis := m.themed().GlyphSet().Ellipsis
	widths := fitWidths(n.naturalWidths(), s.W-cursorGutter)

	headers := make([]string, len(m.Headers))
	for i, h := range m.Headers {
		headers[i] = clip(ansi.Clean(m.Raw, h), widths[i], ellipsis)
	}

	// Window of body rows: the header and divider take two rows.
	body := s.H - 2
	if body < 0 {
		body = 0
	}
	start, end := 0, len(m.Rows)
	if len(m.Rows) > body {
		start = m.cursor - body/2
		if max := len(m.Rows) - body; start > max {
			start = max
		}
		if start < 0 {
			start = 0
		}
		end = start + body
	}
	rows := make([][]string, 0, end-start)
	for _, row := range m.Rows[start:end] {
		cells := make([]string, len(row))
		for i, cell := range row {
			if i < len(widths) {
				cells[i] = clip(ansi.Clean(m.Raw, cell), widths[i], ellipsis)
			} else {
				cells[i] = ansi.Clean(m.Raw, cell)
			}
		}
		rows = append(rows, cells)
	}

	headerStyle := ansi.NewStyle().Bold().Foreground(m.themed().Primary)
	dividerStyle := ansi.NewStyle().Foreground(m.themed().Muted)
	cursorStyle := m.themed().ResolvedStates().Selected.Bold()
	out := widgets.TableRowsWith(headers, rows, headerStyle, dividerStyle, func(i int) ansi.Style {
		if start+i == m.cursor {
			return cursorStyle
		}
		return ansi.Style{}
	}, m.themed())
	return layout.Block(strings.TrimRight(withGutter(out, start, m.cursor), "\n")).Render(s)
}
