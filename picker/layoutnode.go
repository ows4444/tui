package picker

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the list to a layout.Node. Measure reports its natural
// size (widest "> label" by one row per item). Render fits the allotted Size:
// items are cut to the width, and when there are more items than rows it
// shows a window of items containing the cursor item. The Model is not
// changed.
func (m Model) LayoutNode() layout.Node { return listNode{m} }

type listNode struct{ m Model }

func (n listNode) Measure(c layout.Constraints) layout.Size {
	w, h := 0, 0
	for _, r := range n.m.visible() {
		if lw := 2 + ansi.Width(n.m.Items[r.Index].Label); lw > w {
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
	lines := strings.Split(n.m.View(), "\n")
	if len(n.m.Items) == 0 && !n.m.Filter {
		lines = nil
	}
	if len(lines) > s.H {
		// Keep the query line on top, window the items around the cursor.
		head := []string(nil)
		if n.m.Filter {
			head, lines = lines[:1], lines[1:]
		}
		room := s.H - len(head)
		if room < 1 {
			room = 1
		}
		if len(lines) > room {
			row := n.m.rowOf(n.m.visible())
			start := clamp(row-room/2, 0, len(lines)-room)
			lines = lines[start : start+room]
		}
		lines = append(head, lines...)
		if len(lines) > s.H {
			lines = lines[:s.H]
		}
	}
	return layout.Block(strings.Join(lines, "\n")).Render(s)
}
