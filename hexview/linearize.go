package hexview

import (
	"fmt"
	"strconv"
)

// Linearize renders the view as plain text for accessible output (see
// tui.Linearizer): how many bytes there are, where the cursor is and its value,
// and the selection if there is one. No hex dump, colour or cursor marks.
func (m Model) Linearize() string {
	if len(m.Data) == 0 {
		return "Hex view: no data"
	}
	c := m.Cursor()
	out := fmt.Sprintf("Hex view, %d bytes, cursor at offset %d (0x%x), value 0x%02x", len(m.Data), c, c, m.Data[c])
	if b := m.Data[c]; b >= 0x20 && b < 0x7f {
		out += " " + strconv.Quote(string(rune(b)))
	}
	if from, to, ok := m.Selection(); ok && to > from {
		out += fmt.Sprintf(", selected %d bytes from offset %d", to-from+1, from)
	}
	return out
}
