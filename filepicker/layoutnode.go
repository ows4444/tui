package filepicker

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the listing to a layout.Node. Measure reports its
// natural size (widest "> name/" by one row per entry). Render fits the
// allotted Size: names are cut to the width, and when there are more entries
// than rows it shows a window containing the cursor entry. The Model is not
// changed.
func (m Model) LayoutNode() layout.Node { return dirNode{m} }

type dirNode struct{ m Model }

func (n dirNode) Measure(c layout.Constraints) layout.Size {
	w := 0
	for _, e := range n.m.entries {
		lw := 2 + ansi.Width(ansi.Clean(n.m.Raw, e.Name))
		if e.IsDir {
			lw++
		}
		if lw > w {
			w = lw
		}
	}
	return c.Constrain(layout.Size{W: w, H: len(n.m.entries)})
}

func (n dirNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	start, end := layout.WindowRange(len(n.m.entries), n.m.cursor, s.H)
	return layout.Block(n.m.render(start, end)).Render(s)
}
