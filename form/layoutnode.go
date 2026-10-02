package form

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the form to a layout.Node. Measure reports one row per
// field plus one for each error showing, as wide as the widest of them. Render
// draws exactly the Size given: each field's input at the full width (a Secret
// field through its masking node, so its value is never drawn) and each error
// on the row below it. When there are more rows than Size.H, the window
// follows the focused field so it stays visible, rather than clipping to the
// top. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return formNode{m} }

type formNode struct{ m Model }

// child is field i's input as a layout node.
func (n formNode) child(i int) layout.Node { return n.m.inputs[i].layoutNode(n.m.themed()) }

func (n formNode) Measure(c layout.Constraints) layout.Size {
	var size layout.Size
	for i := range n.m.inputs {
		w := n.child(i).Measure(layout.Unconstrained()).W
		size.H++
		if e := n.m.errs[i]; e != "" {
			w = max(w, 2+ansi.Width(e))
			size.H++
		}
		size.W = max(size.W, w)
	}
	return c.Constrain(size)
}

func (n formNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	errStyle := ansi.NewStyle().Foreground(n.m.themed().Error)
	var rows []string
	focusEnd := 0 // index of the last row belonging to the focused field
	for i := range n.m.inputs {
		rows = append(rows, n.child(i).Render(layout.Size{W: s.W, H: 1}))
		if e := n.m.errs[i]; e != "" {
			rows = append(rows, layout.Block("  "+errStyle.Render(e)).Render(layout.Size{W: s.W, H: 1}))
		}
		if i == n.m.ring.Current() {
			focusEnd = len(rows) - 1
		}
	}
	top := 0
	if len(rows) > s.H {
		top = min(max(focusEnd+1-s.H, 0), len(rows)-s.H)
	}
	return layout.Block(strings.Join(rows[top:min(top+s.H, len(rows))], "\n")).Render(s)
}
