package widgets

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Link renders text as a clickable hyperlink via the OSC 8 escape sequence
// (ansi.Hyperlink), underlined and colored with t.Primary so it reads as a
// link even in terminals that ignore OSC 8.
//
// This package has no way to detect whether the attached terminal actually
// supports OSC 8 hyperlinks, so showHref is an always-on visible fallback,
// not a conditional one: when true, the literal href is appended after the
// text in faint styling, guaranteeing the destination is readable even when
// OSC 8 support is absent (or the text is copied out of the terminal).
//
// An empty href disables the hyperlink entirely: text is rendered plainly,
// with no OSC 8 wrapping. The same applies when href fails
// ansi.SafeLinkTarget (control characters, or a scheme other than http,
// https, mailto or file, including relative targets); the unsafe href is
// never emitted, even as the showHref fallback.
func Link(text, href string, showHref bool, t theme.Theme) string {
	if !ansi.SafeLinkTarget(href) {
		return text
	}

	out := ansi.NewStyle().Underline().Foreground(t.Primary).Render(ansi.Hyperlink(text, href))
	if showHref {
		out += " " + ansi.NewStyle().Faint().Render(href)
	}
	return out
}
