package datatable

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// Column describes how one column of the table is shown and sorted. The zero
// value changes nothing. Model.Columns[i] belongs to Headers[i]; a Columns
// slice shorter than Headers leaves the rest as they were.
type Column struct {
	// Align places the text of the column within its width: AlignStart (the
	// default) flush left, AlignEnd flush right (numbers), AlignCenter centred.
	// The header follows.
	Align layout.Align
	// Min and Max bound the column's width in cells; 0 is no bound. A cell wider
	// than Max ends in an ellipsis. With a Source or a very tall table the widths
	// are those of the rows in view.
	Min, Max int
	// Format turns a cell's text into the text shown, such as a byte count into
	// "1.2 MB". Sorting, filtering and SelectedMsg.Cells still use the cell as
	// given. It is called for every cell drawn, so it should be cheap.
	Format func(cell string) string
	// Less, when set, orders two cells of this column for the sort key; the
	// default orders as numbers when both parse and as strings otherwise.
	Less func(a, b string) bool
}

// column returns the settings of column i, the zero Column past the end.
func (m Model) column(i int) Column {
	if i >= 0 && i < len(m.Columns) {
		return m.Columns[i]
	}
	return Column{}
}

// hasColumns reports whether any column changes how the table is drawn.
func (m Model) hasColumns() bool {
	for _, c := range m.Columns {
		if c.Align != layout.AlignStart || c.Min > 0 || c.Max > 0 || c.Format != nil {
			return true
		}
	}
	return false
}

// hasFormat reports whether any column has a Format.
func (m Model) hasFormat() bool {
	for _, c := range m.Columns {
		if c.Format != nil {
			return true
		}
	}
	return false
}

// display is the text shown for a cell of column c: Format applied, then cut to
// Max. It does not clean the text.
func (m Model) display(c int, cell string) string {
	col := m.column(c)
	if col.Format != nil {
		cell = col.Format(cell)
	}
	if col.Max > 0 {
		cell = clip(cell, col.Max, m.themed().GlyphSet().Ellipsis)
	}
	return cell
}

// displayRows applies display to every cell. own is called before the rows are
// copied, so the caller's slices are never written.
func (m Model) displayRows(rows [][]string, own *func()) [][]string {
	if !m.hasFormat() && !m.hasMax() {
		return rows
	}
	(*own)()
	out := make([][]string, len(rows))
	for i, row := range rows {
		cells := make([]string, len(row))
		for c, v := range row {
			cells[c] = m.display(c, v)
		}
		out[i] = cells
	}
	return out
}

func (m Model) hasMax() bool {
	for _, c := range m.Columns {
		if c.Max > 0 {
			return true
		}
	}
	return false
}

// padTo pads s to width cells according to a.
func padTo(s string, width int, a layout.Align) string {
	d := width - ansi.Width(s)
	if d <= 0 {
		return s
	}
	switch a {
	case layout.AlignEnd:
		return spaces(d) + s
	case layout.AlignCenter:
		return spaces(d/2) + s + spaces(d-d/2)
	}
	return s + spaces(d)
}

func spaces(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}

// alignRows pads the cells and the header of every column with an Align or a
// Min to a common width, the widest of its header, its cells in rows and Min,
// so the text lines up whatever the column's alignment. widths, when it is not
// nil, gives the width to use instead (the layout node, which has decided them).
func (m Model) alignRows(headers []string, rows [][]string, widths []int) ([]string, [][]string) {
	var cols []int
	for i := range headers {
		if c := m.column(i); c.Align != layout.AlignStart || c.Min > 0 {
			cols = append(cols, i)
		}
	}
	if len(cols) == 0 {
		return headers, rows
	}
	target := make(map[int]int, len(cols))
	for _, c := range cols {
		if widths != nil && c < len(widths) {
			target[c] = widths[c]
			continue
		}
		w := max(m.column(c).Min, ansi.Width(headers[c]))
		for _, r := range rows {
			if c < len(r) {
				w = max(w, ansi.Width(r[c]))
			}
		}
		target[c] = w
	}
	nh := append([]string(nil), headers...)
	nr := make([][]string, len(rows))
	for i, r := range rows {
		cells := append([]string(nil), r...)
		for _, c := range cols {
			if c < len(cells) {
				cells[c] = padTo(cells[c], target[c], m.column(c).Align)
			}
		}
		nr[i] = cells
	}
	for _, c := range cols {
		nh[c] = padTo(nh[c], target[c], m.column(c).Align)
	}
	return nh, nr
}

// lessFor returns the ordering of column c for the sort key.
func (m Model) lessFor(c int) func(a, b string) bool {
	if l := m.column(c).Less; l != nil {
		return l
	}
	return lessCell
}
