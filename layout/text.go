package layout

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// TextOpt configures a Text node.
type TextOpt func(*textNode)

// WithWrap word-wraps the text to the width the node is given. A word is
// only broken across lines when it is longer than that width. Without it,
// each line is clipped to the width.
func WithWrap() TextOpt { return func(n *textNode) { n.wrap = true } }

// WithStyle applies st to every rendered line, after wrapping so an escape
// sequence is never split.
func WithStyle(st ansi.Style) TextOpt { return func(n *textNode) { n.style = st } }

// WithEllipsis ends a line wider than the width with glyph instead of cutting
// it mid-word, so the reader can tell text was dropped. Pass the theme's
// Glyphs.Ellipsis so it follows the ASCII fallback; an empty glyph uses "~".
// The line, glyph included, is at most the width.
func WithEllipsis(glyph string) TextOpt {
	if glyph == "" {
		glyph = "~"
	}
	return func(n *textNode) { n.ellipsis, n.glyph = true, glyph }
}

// WithAlign places each line within the width: AlignStart (the default) flush
// left, AlignCenter centred (an odd remainder goes after the line) and
// AlignEnd flush right. It does not change what Measure reports.
func WithAlign(a Align) TextOpt { return func(n *textNode) { n.align = a } }

// Text returns a Node for a block of text. With WithWrap, Measure reports
// the wrapped height for the width it may use. Render shows exactly the
// allotted Size, cutting columns and rows beyond it. Empty text measures as
// nothing.
func Text(s string, opts ...TextOpt) Node {
	n := textNode{text: s}
	for _, o := range opts {
		o(&n)
	}
	return n
}

type textNode struct {
	text     string
	style    ansi.Style
	wrap     bool
	ellipsis bool
	glyph    string
	align    Align
}

func (n textNode) lines(width int) []string {
	t := n.text
	if n.wrap && width > 0 {
		t = ansi.WrapStyled(t, width)
	}
	return strings.Split(t, "\n")
}

func (n textNode) Measure(c Constraints) Size {
	if n.text == "" {
		return c.Constrain(Size{})
	}
	width := 0
	if n.wrap && c.MaxW > 0 && c.MaxW < Unbounded {
		width = c.MaxW
	}
	w := 0
	lines := n.lines(width)
	for _, l := range lines {
		if lw := ansi.Width(l); lw > w {
			w = lw
		}
	}
	return c.Constrain(Size{W: w, H: len(lines)})
}

func (n textNode) Render(s Size) string {
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
		if n.ellipsis && ansi.Width(l) > s.W {
			l = ansi.Truncate(l, s.W-ansi.Width(n.glyph)) + n.glyph
		}
		pad := 0
		switch n.align {
		case AlignCenter:
			pad = (s.W - ansi.Width(l)) / 2
		case AlignEnd:
			pad = s.W - ansi.Width(l)
		}
		if l != "" {
			l = n.style.Render(l)
		}
		if pad > 0 {
			l = strings.Repeat(" ", pad) + l
		}
		lines[i] = l
	}
	return Block(strings.Join(lines, "\n")).Render(s)
}
