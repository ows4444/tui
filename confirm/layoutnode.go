package confirm

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the prompt to a layout.Node. When the prompt and both
// options fit on one row it is View exactly; when narrower, the prompt
// word-wraps to the allotted width and the options sit on their own row
// beneath it. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return confirmNode{m} }

type confirmNode struct{ m Model }

// stacked is the wrapped form: prompt text above the options row.
func (n confirmNode) stacked() layout.Node {
	return layout.Column(0,
		layout.FlexChild{Node: layout.StyledText(n.m.Prompt, ansi.Style{}, true)},
		layout.FlexChild{Node: layout.Block(n.m.options())},
	)
}

func (n confirmNode) Measure(c layout.Constraints) layout.Size {
	if v := n.m.View(); c.MaxW >= ansi.Width(v) {
		return layout.Block(v).Measure(c)
	}
	return n.stacked().Measure(c)
}

func (n confirmNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	if v := n.m.View(); ansi.Width(v) <= s.W {
		return layout.Block(v).Render(s)
	}
	return n.stacked().Render(s)
}
