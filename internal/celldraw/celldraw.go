// Package celldraw holds the small helpers the components share when they draw
// straight into a cellbuf.Buffer (their DrawCells methods) instead of building
// a View string for the frame to parse.
package celldraw

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

// StyleID returns the id in buf of the look st gives text, and false when the
// cell grid cannot represent it. The zero ansi.Style is id 0.
func StyleID(buf *cellbuf.Buffer, st ansi.Style) (cellbuf.StyleID, bool) {
	p, err := cellbuf.Parse(st.Render("x"))
	if err != nil {
		return 0, false
	}
	return buf.StyleID(p.Style(p.At(0, 0).Style)), true
}

// Plain reports whether every string is free of escape and other control
// characters, so that SetString (which removes them) writes what View would
// have drawn. A component draws its View string when this is false.
func Plain(ss ...string) bool {
	for _, s := range ss {
		if strings.ContainsFunc(s, func(r rune) bool { return r < ' ' || r == 0x7f }) {
			return false
		}
	}
	return true
}
