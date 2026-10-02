package virtuallist

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
)

// Linearize renders the visible items as plain text for accessible output
// (see tui.Linearizer): a header "List, N items, showing A to B" and one line
// per visible item, "Item I of N: <text>". Only the visible window is
// rendered, so a huge list stays cheap; overscan rows are not included. An
// item that spans several lines has them joined with spaces.
func (m Model) Linearize() string {
	if m.ItemCount <= 0 || m.Height <= 0 || m.RenderItem == nil {
		return "List, empty"
	}
	end := m.offset + m.visibleCount()
	if end > m.ItemCount {
		end = m.ItemCount
	}
	n := strconv.Itoa(m.ItemCount)
	out := []string{"List, " + n + " items, showing " + strconv.Itoa(m.offset+1) + " to " + strconv.Itoa(end)}
	for i := m.offset; i < end; i++ {
		text := strings.Join(strings.Fields(ansi.StripANSI(m.RenderItem(i))), " ")
		out = append(out, "Item "+strconv.Itoa(i+1)+" of "+n+": "+text)
	}
	return strings.Join(out, "\n")
}
