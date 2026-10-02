package commandpalette

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the palette to a layout.Node. Measure reports its
// natural size: the input line and, while the dropdown is open, one row per
// command. Render fits the allotted Size: the input scrolls its value around
// the cursor within the width (as textinput's node does), rows are cut to the
// width, and when the dropdown has more rows than fit it shows a window
// containing the highlighted command beneath the input line. The Model is not
// changed.
func (m Model) LayoutNode() layout.Node { return dropNode{m} }

type dropNode struct{ m Model }

func (n dropNode) Measure(c layout.Constraints) layout.Size {
	lines := strings.Split(n.m.View(), "\n")
	w := 0
	for _, l := range lines {
		if lw := ansi.Width(l); lw > w {
			w = lw
		}
	}
	return c.Constrain(layout.Size{W: w, H: len(lines)})
}

func (n dropNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	m.Input.Width = max(s.W-ansi.Width(m.Input.Prompt)-1, 1)
	lines := strings.Split(m.View(), "\n")
	if len(lines) > s.H {
		// Keep the input line, and a window of the dropdown around the
		// highlighted row beneath it.
		rows := layout.Window(lines[1:], m.highlight, s.H-1)
		lines = append([]string{lines[0]}, rows...)
		if s.H == 1 {
			lines = lines[:1]
		}
	}
	return layout.Block(strings.Join(lines, "\n")).Render(s)
}
