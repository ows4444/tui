package streamtext

import (
	"strings"
	"sync"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

// styleCache memoises the cell form of an ansi.Style (bounded).
var styleCache struct {
	sync.RWMutex
	m map[ansi.Style]cellbuf.Style
}

const styleCacheMax = 256

// cellStyle returns the cell style st renders as: the style a parse of
// st.Render("x") gives to its cell; the default style when the cell grid
// cannot represent it.
func cellStyle(st ansi.Style) cellbuf.Style {
	styleCache.RLock()
	cs, ok := styleCache.m[st]
	styleCache.RUnlock()
	if ok {
		return cs
	}
	if b, err := cellbuf.Parse(st.Render("x")); err == nil {
		cs = b.Style(b.At(0, 0).Style)
	}
	styleCache.Lock()
	if styleCache.m == nil {
		styleCache.m = make(map[ansi.Style]cellbuf.Style)
	}
	if len(styleCache.m) < styleCacheMax {
		styleCache.m[st] = cs
	}
	styleCache.Unlock()
	return cs
}

// DrawCells draws the revealed text and its cursor into the region r of buf,
// showing the same screen as View without building View's string. It makes
// Model a tui.CellDrawer. Each line of the text is drawn with its SGR styling;
// a line the cell grid cannot represent is drawn as plain text. Drawing is
// clipped to r.
func (m Model) DrawCells(buf *cellbuf.Buffer, r cellbuf.Rect) {
	sub := buf.Sub(r)
	if sub.Width() == 0 || sub.Height() == 0 {
		return
	}
	out, done := m.revealed()
	y, x := 0, 0
	for rest, more := out, true; more; y++ {
		line := rest
		if i := strings.IndexByte(rest, '\n'); i >= 0 {
			line, rest = rest[:i], rest[i+1:]
		} else {
			more = false
		}
		if y >= sub.Height() {
			return
		}
		line = strings.TrimSuffix(line, "\r")
		n, err := sub.SetStyled(0, y, line)
		if err != nil {
			n = sub.SetString(0, y, line, 0)
		}
		x = n
	}
	y--
	if c, ok := m.cursorToDraw(done); ok {
		sub.SetString(x, y, c, sub.StyleID(cellStyle(ansi.NewStyle().Foreground(m.themed().Primary))))
	}
}
