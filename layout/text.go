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

// Overflow says which part of a line wider than its width is dropped.
type Overflow int

const (
	// OverflowEnd keeps the start of the line and drops the end. It is what
	// WithEllipsis does.
	OverflowEnd Overflow = iota
	// OverflowStart keeps the end of the line: the file name of a path.
	OverflowStart
	// OverflowMiddle keeps both ends and drops the middle: a long path or
	// identifier whose start and end both say something.
	OverflowMiddle
)

// WithOverflow ends a line wider than the width the way o says, with glyph
// standing where text was dropped (see WithEllipsis; an empty glyph uses "~").
// The line, glyph included, is at most the width.
func WithOverflow(o Overflow, glyph string) TextOpt {
	if glyph == "" {
		glyph = "~"
	}
	return func(n *textNode) { n.ellipsis, n.glyph, n.overflow = true, glyph, o }
}

// WithJustify sets wrapped text flush on both sides: the gaps between words
// grow until each line is as wide as the node, except the last line of a
// paragraph and a line with no gap. It needs WithWrap, and it does not change
// what Measure reports.
func WithJustify() TextOpt { return func(n *textNode) { n.justify = true } }

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
	overflow Overflow
	justify  bool
	align    Align
}

func (n textNode) lines(width int) []string {
	t := n.text
	if n.wrap && width > 0 {
		if n.justify {
			return justifyParagraphs(t, width)
		}
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
			switch n.overflow {
			case OverflowStart:
				l = ansi.TruncateStart(l, s.W, n.glyph)
			case OverflowMiddle:
				l = ansi.TruncateMiddle(l, s.W, n.glyph)
			default:
				l = ansi.Truncate(l, s.W-ansi.Width(n.glyph)) + n.glyph
			}
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

// justifyParagraphs wraps each paragraph of t to width and widens every line of
// it but the last.
func justifyParagraphs(t string, width int) []string {
	var out []string
	for _, para := range strings.Split(t, "\n") {
		wrapped := strings.Split(ansi.WrapStyled(para, width), "\n")
		for i, l := range wrapped {
			if i < len(wrapped)-1 {
				l = ansi.Justify(l, width)
			}
			out = append(out, l)
		}
	}
	return out
}
