package treeview

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the tree to a layout.Node. Measure reports its natural
// size (the widest visible row by one row per visible node). Render fits the
// allotted Size: rows are cut to the width, and when there are more visible
// rows than fit it shows a window of rows containing the cursor row. The
// Model is not changed.
func (m Model) LayoutNode() layout.Node { return treeNode{m} }

type treeNode struct{ m Model }

func (n treeNode) Measure(c layout.Constraints) layout.Size {
	if len(n.m.flatten()) == 0 && !n.m.Filter {
		return c.Constrain(layout.Size{})
	}
	w := 0
	lines := strings.Split(n.m.View(), "\n")
	for _, l := range lines {
		if lw := ansi.Width(l); lw > w {
			w = lw
		}
	}
	return c.Constrain(layout.Size{W: w, H: len(lines)})
}

func (n treeNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	if len(n.m.flatten()) == 0 && !n.m.Filter {
		return layout.Block("").Render(s)
	}
	lines := strings.Split(n.m.View(), "\n")
	if len(lines) > s.H {
		// Keep the query line on top, window the rows under it.
		var head []string
		if n.m.Filter {
			head, lines = lines[:1], lines[1:]
		}
		room := max(s.H-len(head), 1)
		if len(lines) > room {
			cursor := clamp(n.m.cursor, 0, len(lines)-1)
			start := clamp(cursor-room/2, 0, len(lines)-room)
			lines = lines[start : start+room]
		}
		lines = append(head, lines...)
		if len(lines) > s.H {
			lines = lines[:s.H]
		}
	}
	return layout.Block(strings.Join(lines, "\n")).Render(s)
}
