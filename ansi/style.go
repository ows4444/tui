package ansi

import (
	"strconv"
	"strings"
)

// Style is an immutable, chainable SGR (Select Graphic Rendition) builder.
// Every method returns a new Style, so a base style can be reused safely:
//
//	base := ansi.NewStyle().Bold()
//	ok := base.Foreground(ansi.Green)
//	err := base.Foreground(ansi.Red)
type Style struct {
	fg, bg                          Color
	underlineColor                  Color
	bold, faint, italic, underline  bool
	blink, reverse, strike, conceal bool
	overline                        bool
	underlineStyle                  UnderlineStyle
	link                            string // a SafeLinkTarget, or ""
}

// UnderlineStyle selects the shape of an underline drawn via
// Style.UnderlineStyle, using the extended "4:n" SGR sub-parameter that
// Kitty, WezTerm, iTerm2 and recent VTE/Ghostty support. A terminal that
// doesn't understand the colon sub-parameter form typically falls back to
// an ordinary straight underline instead of ignoring it outright.
type UnderlineStyle uint8

const (
	// UnderlineCurly draws a wavy underline (SGR 4:3), the shape commonly
	// used for spell-check/lint-style annotations.
	UnderlineCurly UnderlineStyle = 3
	// UnderlineDotted draws a dotted underline (SGR 4:4).
	UnderlineDotted UnderlineStyle = 4
	// UnderlineDashed draws a dashed underline (SGR 4:5).
	UnderlineDashed UnderlineStyle = 5
)

// NewStyle returns an empty Style with no attributes set.
func NewStyle() Style { return Style{} }

// Foreground sets the text color.
func (s Style) Foreground(c Color) Style { s.fg = c; return s }

// Background sets the background color.
func (s Style) Background(c Color) Style { s.bg = c; return s }

// Bold enables bold text.
func (s Style) Bold() Style { s.bold = true; return s }

// Faint enables dim/faint text.
func (s Style) Faint() Style { s.faint = true; return s }

// Italic enables italic text.
func (s Style) Italic() Style { s.italic = true; return s }

// Underline enables underlined text.
func (s Style) Underline() Style { s.underline = true; return s }

// UnderlineStyle enables underlined text drawn in the given shape (curly,
// dotted, dashed) instead of a plain straight line. It implies Underline,
// so calling it alone is enough — no need to also call Underline().
func (s Style) UnderlineStyle(style UnderlineStyle) Style {
	s.underline = true
	s.underlineStyle = style
	return s
}

// UnderlineColor sets the underline's color independently of the text's
// foreground color (SGR 58), so an underline can, for example, mark an
// error in red under text that otherwise renders in the default color.
// It has no visible effect unless the style is also underlined (via
// Underline or UnderlineStyle).
func (s Style) UnderlineColor(c Color) Style { s.underlineColor = c; return s }

// Blink enables blinking text.
func (s Style) Blink() Style { s.blink = true; return s }

// Reverse swaps the foreground and background colors.
func (s Style) Reverse() Style { s.reverse = true; return s }

// Strikethrough enables struck-through text.
func (s Style) Strikethrough() Style { s.strike = true; return s }

// Conceal hides the text (still selectable, invisible to most terminals).
func (s Style) Conceal() Style { s.conceal = true; return s }

// Overline draws a line above the text (SGR 53). Reset clears it.
func (s Style) Overline() Style { s.overline = true; return s }

// Link makes Render and Span wrap the styled text in an OSC 8 hyperlink to url,
// one link per non-empty line. A url that fails SafeLinkTarget is dropped, so
// the text renders without OSC 8, exactly as Hyperlink does.
func (s Style) Link(url string) Style {
	if SafeLinkTarget(url) {
		s.link = url
	} else {
		s.link = ""
	}
	return s
}

// linkEach renders text with f on a copy of s that has no link, and wraps each
// non-empty line of the result in an OSC 8 hyperlink.
func (s Style) linkEach(text string, f func(Style, string) string) string {
	url := s.link
	s.link = ""
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = Hyperlink(f(s, line), url)
		}
	}
	return strings.Join(lines, "\n")
}

// sequence returns the SGR escape sequence for s, or "" if s sets nothing.
func (s Style) sequence() string {
	var buf [96]byte
	return string(s.appendSequence(buf[:0]))
}

// appendParam appends one SGR parameter, preceded by ';' unless it is the
// first (start is the length of dst just after the CSI introducer).
func appendParam(dst []byte, start int, p string) []byte {
	if len(dst) > start {
		dst = append(dst, ';')
	}
	return append(dst, p...)
}

// appendSequence appends s's escape sequence to dst and returns it, or dst
// unchanged if s sets nothing. Parameters keep a fixed order: bold, faint,
// italic, underline (4 or 4:n), blink, reverse, strikethrough, conceal,
// overline, then foreground, background and underline colours.
func (s Style) appendSequence(dst []byte) []byte {
	base := len(dst)
	dst = append(dst, CSI...)
	start := len(dst)
	if s.bold {
		dst = appendParam(dst, start, "1")
	}
	if s.faint {
		dst = appendParam(dst, start, "2")
	}
	if s.italic {
		dst = appendParam(dst, start, "3")
	}
	if s.underline {
		if s.underlineStyle != 0 {
			dst = appendParam(dst, start, "4:")
			dst = strconv.AppendInt(dst, int64(s.underlineStyle), 10)
		} else {
			dst = appendParam(dst, start, "4")
		}
	}
	if s.blink {
		dst = appendParam(dst, start, "5")
	}
	if s.reverse {
		dst = appendParam(dst, start, "7")
	}
	if s.strike {
		dst = appendParam(dst, start, "9")
	}
	if s.conceal {
		dst = appendParam(dst, start, "8")
	}
	if s.overline {
		dst = appendParam(dst, start, "53")
	}
	if s.fg != nil {
		dst = appendParam(dst, start, "")
		dst = appendSGRColor(dst, s.fg, slotFg)
	}
	if s.bg != nil {
		dst = appendParam(dst, start, "")
		dst = appendSGRColor(dst, s.bg, slotBg)
	}
	if s.underlineColor != nil {
		dst = appendParam(dst, start, "")
		dst = appendSGRColor(dst, s.underlineColor, slotUl)
	}
	if len(dst) == start {
		return dst[:base]
	}
	return append(dst, 'm')
}

// Render wraps text in this style's escape sequence, resetting afterward.
// Multi-line text is styled per line — each non-empty line gets its own
// opening sequence and Reset, and empty lines stay bare — so any single
// line remains styled when repainted on its own (as Program.render's line
// diff does).
//
// Render's Reset is a full SGR reset (Style has no save/restore stack), so
// nesting one Style.Render call around text that already contains another
// Style's Render output does not compose: the inner Reset fires before the
// outer text that follows it, clobbering the outer style for everything
// after the nested content, even on a row that's supposed to stay
// highlighted throughout (e.g. a cursor row embedding a pre-styled swatch).
// Compute one combined Style for a run that needs to look like nested
// styling, rather than composing two independently-Render-ed strings.
func (s Style) Render(text string) string {
	if s.link != "" {
		return s.linkEach(text, Style.Render)
	}
	// The whole escape sequence is built in a stack buffer, and the result in
	// one pre-sized Builder, so a call allocates once (the result string).
	var buf [96]byte
	seq := s.appendSequence(buf[:0])
	if len(seq) == 0 {
		return text
	}
	if strings.IndexByte(text, '\n') < 0 {
		var b strings.Builder
		b.Grow(len(seq) + len(text) + len(Reset))
		b.Write(seq)
		b.WriteString(text)
		b.WriteString(Reset)
		return b.String()
	}

	// Multi-line: each non-empty line is wrapped on its own, so the styling
	// never bleeds across the terminal's line boundaries. Size the result
	// exactly first, then fill it.
	wrapped := 0
	for rest := text; ; {
		i := strings.IndexByte(rest, '\n')
		line := rest
		if i >= 0 {
			line = rest[:i]
		}
		if line != "" {
			wrapped++
		}
		if i < 0 {
			break
		}
		rest = rest[i+1:]
	}
	var b strings.Builder
	b.Grow(len(text) + wrapped*(len(seq)+len(Reset)))
	for rest := text; ; {
		i := strings.IndexByte(rest, '\n')
		line := rest
		if i >= 0 {
			line = rest[:i]
		}
		if line != "" {
			b.Write(seq)
			b.WriteString(line)
			b.WriteString(Reset)
		}
		if i < 0 {
			break
		}
		b.WriteByte('\n')
		rest = rest[i+1:]
	}
	return b.String()
}
