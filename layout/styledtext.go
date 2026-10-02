package layout

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// StyledText returns a Node for a block of text. When wrap is true the text is
// word-wrapped (ansi.Wrap) to whatever width the node is given, and Measure
// reports the wrapped height for the width it may use; the style, if any, is
// applied to each line after wrapping so an escape sequence is never split.
// Render shows exactly the allotted Size, cutting rows beyond the height.
// Empty text measures as nothing.
func StyledText(text string, style ansi.Style, wrap bool) Node {
	return styledTextNode{text: text, style: style, wrap: wrap}
}

type styledTextNode struct {
	text  string
	style ansi.Style
	wrap  bool
}

func (n styledTextNode) lines(width int) []string {
	t := n.text
	if n.wrap && width > 0 {
		t = ansi.WrapStyled(t, width)
	}
	return strings.Split(t, "\n")
}

func (n styledTextNode) Measure(c Constraints) Size {
	if n.text == "" {
		return c.Constrain(Size{})
	}
	width := 0
	if n.wrap && c.MaxW > 0 && c.MaxW < Unbounded {
		width = c.MaxW
	}
	lines := n.lines(width)
	w := 0
	for _, l := range lines {
		if lw := ansi.Width(l); lw > w {
			w = lw
		}
	}
	return c.Constrain(Size{W: w, H: len(lines)})
}

func (n styledTextNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	if n.text == "" {
		return Block("").Render(s)
	}
	lines := n.lines(s.W)
	if len(lines) > s.H {
		lines = lines[:s.H]
	}
	for i, l := range lines {
		if l != "" {
			lines[i] = n.style.Render(l)
		}
	}
	return Block(strings.Join(lines, "\n")).Render(s)
}

// ViewFunc returns a Node that shows whatever f returns each time it is measured
// or rendered, cut to the allotted Size. It is for widgets whose View changes
// with time (a clock) or state, where capturing the string once would go
// stale.
func ViewFunc(f func() string) Node { return viewFuncNode{f} }

type viewFuncNode struct{ f func() string }

func (n viewFuncNode) Measure(c Constraints) Size {
	return Block(n.f()).Measure(c)
}

func (n viewFuncNode) Render(s Size) string {
	return Block(n.f()).Render(s)
}
