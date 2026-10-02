package accordion

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the accordion to a layout.Node. Measure reports its
// natural size (widest row by one row per title plus one per content line of
// each expanded section). Render fits the allotted Size: rows are cut to the
// width, and when the accordion is taller than the space it shows a window
// containing the cursor section's title. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return accordionNode{m} }

type accordionNode struct{ m Model }

func (n accordionNode) Measure(c layout.Constraints) layout.Size {
	if len(n.m.Sections) == 0 {
		return c.Constrain(layout.Size{})
	}
	lines := strings.Split(n.m.View(), "\n")
	w := 0
	for _, l := range lines {
		if lw := ansi.Width(l); lw > w {
			w = lw
		}
	}
	return c.Constrain(layout.Size{W: w, H: len(lines)})
}

// cursorLine is the row of the cursor section's title in View's output.
func (n accordionNode) cursorLine() int {
	row := 0
	for i, s := range n.m.Sections {
		if i == n.m.cursor {
			return row
		}
		row++
		if n.m.expanded[i] && s.Content != "" {
			row += strings.Count(s.Content, "\n") + 1
		}
	}
	return 0
}

func (n accordionNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	if len(n.m.Sections) == 0 {
		return layout.Block("").Render(s)
	}
	lines := layout.Window(strings.Split(n.m.View(), "\n"), n.cursorLine(), s.H)
	return layout.Block(strings.Join(lines, "\n")).Render(s)
}
