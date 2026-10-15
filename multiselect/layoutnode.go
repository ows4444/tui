package multiselect

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the list to a layout.Node. Measure reports its natural
// size (widest "> [x] label" by one row per item). Render fits the allotted
// Size: items are cut to the width, and when there are more items than rows
// it shows a window containing the cursor item. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return listNode{m} }

type listNode struct{ m Model }

func (n listNode) Measure(c layout.Constraints) layout.Size {
	w, h := 0, 0
	for _, r := range n.m.visible() {
		if lw := 2 + 3 + 1 + ansi.Width(n.m.Items[r.Index].Label); lw > w {
			w = lw
		}
		h++
	}
	if n.m.Filter {
		h++ // the query line
		if h == 1 {
			h = 2 // "no matches"
		}
		if qw := 2 + ansi.Width(n.m.query); qw > w {
			w = qw
		}
	}
	return c.Constrain(layout.Size{W: w, H: h})
}

func (n listNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	var lines []string
	if len(n.m.Items) > 0 || n.m.Filter {
		all := strings.Split(n.m.View(), "\n")
		if n.m.Filter && len(all) > 1 && s.H > 1 {
			// Keep the query line on top, window the items under it.
			row := n.m.rowOf(n.m.visible())
			lines = append([]string{all[0]}, layout.Window(all[1:], max(row, 0), s.H-1)...)
		} else {
			lines = layout.Window(all, n.m.cursor, s.H)
		}
	}
	return layout.Block(strings.Join(lines, "\n")).Render(s)
}
