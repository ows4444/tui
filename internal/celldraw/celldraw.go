// Package celldraw holds the small helpers the components share when they draw
// straight into a cellbuf.Buffer (their DrawCells methods) instead of building
// a View string for the frame to parse.
package celldraw

import (
	"strings"
	"sync"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

// parse is cellbuf.Parse; a test replaces it to reach the failure branch.
var parse = cellbuf.Parse

// styleCache remembers the cell style each ansi.Style parses to, so a widget
// that asks every frame does not render and parse a probe string every time. It
// is bounded: past styleCacheMax entries a conversion is simply not stored.
var styleCache struct {
	sync.RWMutex
	m map[ansi.Style]cellbuf.Style
}

const styleCacheMax = 512

// StyleID returns the id in buf of the look st gives text, and false when the
// cell grid cannot represent it. The zero ansi.Style is id 0.
func StyleID(buf *cellbuf.Buffer, st ansi.Style) (cellbuf.StyleID, bool) {
	styleCache.RLock()
	cs, ok := styleCache.m[st]
	styleCache.RUnlock()
	if ok {
		return buf.StyleID(cs), true
	}
	p, err := parse(st.Render("x"))
	if err != nil {
		return 0, false
	}
	cs = p.Style(p.At(0, 0).Style)
	styleCache.Lock()
	if styleCache.m == nil {
		styleCache.m = make(map[ansi.Style]cellbuf.Style)
	}
	if len(styleCache.m) < styleCacheMax {
		styleCache.m[st] = cs
	}
	styleCache.Unlock()
	return buf.StyleID(cs), true
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
