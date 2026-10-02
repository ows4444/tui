package colorpicker

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the picker to a layout.Node: the swatches keep their
// room and the hex input scrolls its value in whatever width is left, all cut
// to the allotted Size. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return pickerNode{m} }

type pickerNode struct{ m Model }

func (n pickerNode) swatchesWidth() int {
	if len(n.m.Palette) == 0 {
		return 0
	}
	return len(n.m.Palette)*ansi.Width(swatch) + (len(n.m.Palette) - 1)
}

func (n pickerNode) Measure(c layout.Constraints) layout.Size {
	return layout.Block(n.m.View()).Measure(c)
}

func (n pickerNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	m.HexInput.Width = max(s.W-n.swatchesWidth()-2-ansi.Width(m.HexInput.Prompt)-1, 1)
	return layout.Block(m.View()).Render(s)
}
