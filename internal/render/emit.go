package render

import (
	"strconv"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
)

// cellEmitter builds one frame's bytes and tracks the cursor and pen.
type cellEmitter struct {
	c        *Cells
	out      []byte
	width    int
	row, col int // col < 0: unknown (pending wrap or after a fresh region)
	pen      uint16
}

func (e *cellEmitter) num(n int) { e.out = strconv.AppendInt(e.out, int64(n), 10) }

func (e *cellEmitter) csi(n int, final byte) {
	e.out = append(e.out, 0x1b, '[')
	e.num(n)
	e.out = append(e.out, final)
}

// moveRow moves down to row t. Rows the previous frame did not have are
// reached with CR LF (which also scrolls, as the line renderer does).
func (e *cellEmitter) moveRow(t int, exists bool) {
	d := t - e.row
	if d <= 0 {
		return
	}
	if !exists || d <= 2 {
		for ; d > 0; d-- {
			e.out = append(e.out, '\r', '\n')
		}
		e.col = 0
	} else {
		e.csi(d, 'B')
	}
	e.row = t
}

func (e *cellEmitter) moveCol(t int) {
	switch {
	case e.col == t:
		return
	case t == 0:
		e.out = append(e.out, '\r')
	case e.col >= 0 && t > e.col:
		if t-e.col == 1 {
			e.out = append(e.out, 0x1b, '[', 'C')
		} else {
			e.csi(t-e.col, 'C')
		}
	case e.col >= 0 && t < e.col:
		if e.col-t == 1 {
			e.out = append(e.out, 0x1b, '[', 'D')
		} else {
			e.csi(e.col-t, 'D')
		}
	default:
		e.csi(t+1, 'G')
	}
	e.col = t
}

func appendSGRColor(dst []byte, c cellColor, fg bool) []byte {
	kind, v := int(c>>24), int(c&0xffffff)
	base := 40
	ext := 48
	bright := 100
	if fg {
		base, ext, bright = 30, 38, 90
	}
	switch kind {
	case 0:
		if fg {
			return strconv.AppendInt(dst, 39, 10)
		}
		return strconv.AppendInt(dst, 49, 10)
	case colBasic:
		return strconv.AppendInt(dst, int64(base+v), 10)
	case colBright:
		return strconv.AppendInt(dst, int64(bright+v), 10)
	case col256:
		dst = strconv.AppendInt(dst, int64(ext), 10)
		dst = append(dst, ";5;"...)
		return strconv.AppendInt(dst, int64(v), 10)
	default:
		dst = strconv.AppendInt(dst, int64(ext), 10)
		dst = append(dst, ";2;"...)
		dst = strconv.AppendInt(dst, int64(v>>16), 10)
		dst = append(dst, ';')
		dst = strconv.AppendInt(dst, int64(v>>8&0xff), 10)
		dst = append(dst, ';')
		return strconv.AppendInt(dst, int64(v&0xff), 10)
	}
}

// setPen emits the SGR that takes the terminal from the current pen to id:
// just the additions when nothing has to be turned off, else a reset plus
// the full state.
func (e *cellEmitter) setPen(id uint16) {
	if id == e.pen {
		return
	}
	c := e.c
	if e.pen < transN && id < transN {
		if c.trans == nil {
			c.trans = make([]string, transN*transN)
		}
		slot := int(e.pen)*transN + int(id)
		if t := c.trans[slot]; t != "" {
			e.out = append(e.out, t...)
			e.pen = id
			return
		}
		start := len(e.out)
		e.penChange(id)
		c.trans[slot] = string(e.out[start:]) // "" (no change to write) just stays uncached
		return
	}
	e.penChange(id)
}

// penChange writes the bytes that take the terminal from the current pen to
// style id, and makes id the pen.
func (e *cellEmitter) penChange(id uint16) {
	from, to := e.c.styles[e.pen], e.c.styles[id]
	e.pen = id
	e.setSGR(from, to)
	if from.link != to.link {
		if to.link == 0 {
			e.out = append(e.out, "\x1b]8;;\x1b\\"...)
		} else {
			e.out = append(e.out, e.c.links[to.link]...)
		}
	}
}

// setSGR emits the SGR part of the pen change.
func (e *cellEmitter) setSGR(from, to cellStyle) {
	from.link, to.link = 0, 0
	if from == to {
		return
	}
	if to == (cellStyle{}) {
		e.out = append(e.out, ansi.Reset...)
		return
	}
	e.out = append(e.out, 0x1b, '[')
	first := true
	sep := func() {
		if !first {
			e.out = append(e.out, ';')
		}
		first = false
	}
	if from.attrs&^to.attrs != 0 || from.ul != 0 && from.ul != to.ul {
		// Something must be switched off: start over from the default.
		from = cellStyle{}
		sep()
		e.out = append(e.out, '0')
	}
	for b, code := range sgrSetCodes {
		if to.attrs&^from.attrs&(1<<b) != 0 && !(code == 4 && to.ul != 0) {
			sep()
			e.out = strconv.AppendInt(e.out, int64(code), 10)
		}
	}
	if to.ul != from.ul {
		sep()
		e.out = append(e.out, '4', ':', '0'+to.ul)
	}
	if to.fg != from.fg {
		sep()
		e.out = appendSGRColor(e.out, to.fg, true)
	}
	if to.bg != from.bg {
		sep()
		e.out = appendSGRColor(e.out, to.bg, false)
	}
	e.out = append(e.out, 'm')
}

// appendCell writes a cell's characters.
func (e *cellEmitter) appendCell(cl cell) {
	switch {
	case cl.ch < utf8.RuneSelf:
		e.out = append(e.out, byte(cl.ch))
	case cl.ch&clusterFlag != 0:
		e.out = append(e.out, e.c.clusters[cl.ch&^clusterFlag]...)
	default:
		e.out = utf8.AppendRune(e.out, rune(cl.ch)) // #nosec G115 -- without clusterFlag, ch is a rune and fits
	}
}

// writeCells writes n[s:e) at the cursor, which must already be at column s.
func (e *cellEmitter) writeCells(n []cell, s, end int) {
	if end > len(n) {
		// Past the row's end are the blank cells of a longer previous row.
		for k := s; k < end; k++ {
			cl := cellAt(n, k)
			if cl.w == 0 {
				continue
			}
			e.setPen(cl.st)
			e.appendCell(cl)
		}
	} else {
		for _, cl := range n[s:end] {
			if cl.w == 0 {
				continue
			}
			if cl.st != e.pen {
				e.setPen(cl.st)
			}
			e.appendCell(cl)
		}
	}
	e.col = end
	if e.width > 0 && end >= e.width {
		e.col = -1 // pending wrap: the cursor position is not reliable
	}
}

func (e *cellEmitter) eraseFrom(col int) {
	e.moveCol(col)
	e.setPen(0)
	e.out = append(e.out, 0x1b, '[', 'K')
}

// writeOpaque re-sends a row's opaque segments, each at its anchor column.
// They are zero-width, so the cursor does not move past them.
func (e *cellEmitter) writeOpaque(opq []opaqueSeg) {
	for _, sg := range opq {
		col := int(sg.col)
		if e.width > 0 && col >= e.width {
			col = e.width - 1
		}
		e.moveCol(col)
		if isSixel(sg.seq) {
			// A Sixel image moves the cursor as it draws; save and restore it so
			// the segment stays zero-width like every other opaque string.
			e.out = append(e.out, "\x1b7"...)
			e.out = append(e.out, sg.seq...)
			e.out = append(e.out, "\x1b8"...)
			continue
		}
		e.out = append(e.out, sg.seq...)
	}
}

// isSixel reports whether seq is a Sixel image: a DCS string whose final byte
// before the data is q ("ESC P" params "q").
func isSixel(seq string) bool {
	if len(seq) < 3 || seq[0] != 0x1b || seq[1] != 'P' {
		return false
	}
	for i := 2; i < len(seq); i++ {
		switch c := seq[i]; {
		case c >= '0' && c <= '9' || c == ';':
		default:
			return c == 'q'
		}
	}
	return false
}

// KittyDelete returns the kitty graphics command that deletes every placement
// of image id and frees its data, with the terminal's reply suppressed.
func KittyDelete(id uint32) string {
	return "\x1b_Ga=d,d=I,i=" + strconv.FormatUint(uint64(id), 10) + ",q=2\x1b\\"
}
