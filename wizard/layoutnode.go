package wizard

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// LayoutNode adapts the step bar to a layout.Node, drawn with theme t (as
// View is). Measure reports its natural size, one row. When the bar is wider
// than the allotted width, Render scrolls it so the current step is visible.
// The Model is not changed.
func (m Model) LayoutNode(t theme.Theme) layout.Node { return stepsNode{m, t} }

type stepsNode struct {
	m Model
	t theme.Theme
}

func (n stepsNode) Measure(c layout.Constraints) layout.Size {
	if len(n.m.titles) == 0 {
		return c.Constrain(layout.Size{})
	}
	return layout.Block(n.m.View(n.t)).Measure(c)
}

func (n stepsNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	bar := n.m.View(n.t)
	if len(n.m.titles) > 0 && ansi.Width(bar) > s.W {
		// Each step is a two-column marker and its title; " → " (3 columns)
		// separates steps.
		cur := clamp(n.m.current, 0, len(n.m.titles)-1)
		start := 0
		for i := 0; i < cur; i++ {
			start += 2 + ansi.Width(n.m.titles[i]) + 3
		}
		end := start + 2 + ansi.Width(n.m.titles[cur])
		offset := end - s.W
		if offset > start {
			offset = start
		}
		if offset > 0 {
			bar = ansi.TrimLeftWidth(bar, offset)
		}
	}
	return layout.Block(bar).Render(s)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
