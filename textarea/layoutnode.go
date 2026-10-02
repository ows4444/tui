package textarea

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the textarea to a layout.Node. Measure reports the text's
// natural size (widest line by number of lines; a placeholder's width when
// empty). Render fits the allotted Size: it uses the width in place of the
// Model's Width (so long lines scroll horizontally around the cursor, as View
// does when Width is set) and, when there are more lines than rows, shows a
// window of lines that contains the cursor line. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return textareaNode{m} }

type textareaNode struct{ m Model }

func (n textareaNode) Measure(c layout.Constraints) layout.Size {
	if n.m.value.len() == 0 {
		return c.Constrain(layout.Size{W: ansi.Width(n.m.Placeholder), H: 1})
	}
	lines := strings.Split(n.m.value.String(), "\n")
	w := 0
	for _, l := range lines {
		if lw := ansi.Width(l); lw > w {
			w = lw
		}
	}
	return c.Constrain(layout.Size{W: w, H: len(lines)})
}

func (n textareaNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	m.Width = s.W
	lines := strings.Split(m.View(), "\n")
	if len(lines) > s.H {
		cursorLine, _ := m.lineCol()
		start := cursorLine - s.H/2
		if max := len(lines) - s.H; start > max {
			start = max
		}
		if start < 0 {
			start = 0
		}
		lines = lines[start : start+s.H]
	}
	return layout.Block(strings.Join(lines, "\n")).Render(s)
}
