package streamtext

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// LayoutNode adapts the text to a layout.Node. Measure reports the full text's
// natural size (its longest line by its line count). Render word-wraps the
// text to the allotted width, using it in place of Width (leaving room for the
// cursor, as View does), shows as much of the revealed text as fits, and keeps
// the newest lines when there are more than rows. The Model is not changed.
func (m Model) LayoutNode() layout.Node { return textNode{m} }

type textNode struct{ m Model }

func (n textNode) Measure(c layout.Constraints) layout.Size {
	d := n.m.display()
	if d == "" {
		return c.Constrain(layout.Size{})
	}
	w, h := 0, 1
	line := 0
	for i := 0; i <= len(d); i++ {
		if i == len(d) || d[i] == '\n' {
			if lw := ansi.Width(d[line:i]); lw > w {
				w = lw
			}
			if i < len(d) {
				h++
			}
			line = i + 1
		}
	}
	return c.Constrain(layout.Size{W: w, H: h})
}

func (n textNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	m := n.m
	m.Width = s.W
	lines := splitLines(m.View())
	if len(lines) > s.H {
		lines = lines[len(lines)-s.H:] // streaming text: keep the newest
	}
	return layout.Block(joinLines(lines)).Render(s)
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return out
}

func joinLines(lines []string) string {
	s := ""
	for i, l := range lines {
		if i > 0 {
			s += "\n"
		}
		s += l
	}
	return s
}
