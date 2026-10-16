package layout

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// RichText returns a Node for styled text held as data (ansi.Text). When wrap is
// true the text is wrapped to whatever width the node is given (Text.Wrap), and
// Measure reports the wrapped height for the width it may use; otherwise each
// line is cut to the width. Each span keeps its own style across the breaks, and
// nothing is parsed: the lines are measured and cut as spans. Render shows
// exactly the allotted Size, cutting rows beyond the height. Empty text measures
// as nothing.
func RichText(t ansi.Text, wrap bool) Node { return richTextNode{t: t, wrap: wrap} }

type richTextNode struct {
	t    ansi.Text
	wrap bool
}

func (n richTextNode) lines(width int) []ansi.Text {
	if n.wrap && width > 0 {
		return n.t.Wrap(width)
	}
	if width > 0 {
		return n.t.Truncate(width).Lines()
	}
	return n.t.Lines()
}

func (n richTextNode) Measure(c Constraints) Size {
	if len(n.t) == 0 {
		return c.Constrain(Size{})
	}
	width := 0
	if c.MaxW > 0 && c.MaxW < Unbounded {
		width = c.MaxW
	}
	lines := n.lines(width)
	w := 0
	for _, l := range lines {
		w = max(w, l.Width())
	}
	return c.Constrain(Size{W: w, H: len(lines)})
}

func (n richTextNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	if len(n.t) == 0 {
		return Block("").Render(s)
	}
	lines := n.lines(s.W)
	if len(lines) > s.H {
		lines = lines[:s.H]
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.String()
	}
	return Block(strings.Join(out, "\n")).Render(s)
}
