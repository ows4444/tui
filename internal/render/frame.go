package render

import "github.com/ows4444/tui/ansi"

// Frame is one frame for Cells.Frame.
type Frame struct {
	Lines         []string      // the view, one string per row
	Max           int           // rows to draw: max(len(Lines), rows of the previous frame)
	LazyFit       bool          // Lines are not yet fitted to Width; see Cells.Frame
	Width, Height int           // terminal size; Height <= 0 when unknown
	PrevRows      int           // rows the caller recorded for the previous frame; 0 for none
	RegionTop     func() string // moves the cursor to the top of the live region
	FitLine       func(string) string
	Measurer      ansi.Measurer // how the terminal draws grapheme clusters; zero follows the process default
	// Grid, when non-nil, is drawn instead of Lines: GridRows rows of Width
	// columns, with no string parsed. Rows past GridRows (up to Max) are blank.
	Grid     GridSource
	GridRows int
	// HardwareScroll says the region's first row is the screen's first row (an
	// alternate screen), so a block of rows that moved can be scrolled by the
	// terminal instead of rewritten. Lines frames only; ignored with a Grid.
	HardwareScroll bool
}

// Stats describes a drawn frame, for the frame log.
type Stats struct {
	Rows, Changed int
	Full          bool
	// Fallbacks lists the rows drawn with the line strategy this frame (rows
	// the grid could not represent) and why; nil when there were none. The
	// slice is reused by the next Frame.
	Fallbacks []RowFallback
}

// Frame draws in.Lines (padded to in.Max rows) as the change from the previous
// frame. ok is false when this frame must be drawn by the line renderer
// instead; nothing has been recorded then, and Reason says why. With
// in.LazyFit the lines are not yet fitted to the terminal: this fits each one
// it cannot show to be unchanged in width with in.FitLine, in place.
func (c *Cells) Frame(in Frame) (string, Stats, bool) {
	full := in.PrevRows == 0
	c.reason = ""
	c.wide = false
	c.fallbacks = c.fallbacks[:0]
	if in.Measurer != c.measure {
		// Cell widths and cached rows were computed under the old measurer.
		c.measure = in.Measurer
		c.havePrev = false
	}
	if in.Height > 0 && in.Max > in.Height {
		c.havePrev = false
		c.reason = "view_taller_than_terminal"
		return "", Stats{}, false
	}
	// A block of rows that only moved is scrolled by the terminal; the previous
	// frame's rows move with it, so the row cache and the diff below see them in
	// their new places.
	mv, scrolled := c.detectScroll(in)
	if scrolled {
		c.applyScroll(mv)
	}
	fr := c.frameRows(in.Max)
	if !c.buildRows(in, fr) {
		c.havePrev = false
		return "", Stats{}, false
	}
	c.placedNew = c.placedNew[:0]
	for i := 0; i < fr.max; i++ {
		for _, sg := range fr.curInfo[i].opq {
			if sg.kid != 0 {
				c.placedNew = append(c.placedNew, placement{sg.kid, i, int(sg.col)})
			}
		}
	}

	havePrev := c.havePrev && in.PrevRows > 0 && len(c.prev) == in.PrevRows
	prev := c.prev
	e := &cellEmitter{c: c, out: c.out[:0], width: in.Width}
	e.out = append(e.out, in.RegionTop()...)
	if !havePrev {
		e.out = append(e.out, ansi.Reset...) // the pen is unknown
	}
	if scrolled {
		e.out = mv.appendSeq(e.out) // homes the cursor: row 0, column 0 is where it already was
	}
	c.deleteGonePlacements(e)
	changed := 0
	for i := 0; i < fr.max; i++ {
		changed += c.drawRow(e, fr, prev, havePrev, i)
	}
	e.moveRow(fr.max-1, true)
	e.setPen(0)

	c.out = e.out
	c.placed, c.placedNew = c.placedNew, c.placed
	c.width = in.Width
	c.prev, c.cur = fr.cur, prev
	c.prevInfo, c.curInfo = fr.curInfo, c.prevInfo
	c.prevSrc, c.curSrc = fr.curSrc, c.prevSrc
	c.havePrev = true
	c.gridPrev = in.Grid != nil
	return string(e.out), Stats{Rows: fr.max, Changed: changed, Full: full, Fallbacks: c.fallbacks}, true
}

// frameRows is the per-row state of the frame being built: views of the
// Cells' reusable slices, sized to the frame's rows.
type frameRows struct {
	max     int
	cur     [][]cell
	curSrc  []string
	same    []bool // the row's source line is unchanged since the previous frame
	patched []bool // the row was updated in place from the previous frame's
	curInfo []rowInfo
}

// frameRows sizes the reusable slices to max rows.
func (c *Cells) frameRows(max int) *frameRows {
	for len(c.cur) < max {
		c.cur = append(c.cur, nil)
	}
	if cap(c.curSrc) < max {
		c.curSrc = make([]string, max)
	}
	// A row whose source line is unchanged since the previous frame keeps its
	// parsed cells (shared with prev, never mutated) instead of being parsed.
	if cap(c.same) < max {
		c.same = make([]bool, max)
	}
	for len(c.patches) < max {
		c.patches = append(c.patches, nil)
		c.patchPos = append(c.patchPos, nil)
	}
	for len(c.curInfo) < max {
		c.curInfo = append(c.curInfo, rowInfo{})
	}
	if cap(c.patched) < max {
		c.patched = make([]bool, max)
	}
	c.fr = frameRows{
		max: max, cur: c.cur[:max], curSrc: c.curSrc[:max], same: c.same[:max],
		patched: c.patched[:max], curInfo: c.curInfo[:max],
	}
	return &c.fr
}

// buildRows fills in the cells of every row, parsing a line only when it
// changed. It reports false when the frame must be drawn by the line renderer.
func (c *Cells) buildRows(in Frame, fr *frameRows) bool {
	canReuse := !c.noCache && c.havePrev && len(c.prev) == len(c.prevSrc) && !c.gridPrev && in.Grid == nil
	for i := 0; i < fr.max; i++ {
		var line string
		if i < len(in.Lines) {
			line = in.Lines[i]
		}
		fr.curSrc[i] = line
		fr.same[i] = false
		fr.patched[i] = false
		buf := fr.cur[i]
		if i < len(c.prev) && cap(buf) > 0 && cap(c.prev[i]) > 0 && &buf[:1][0] == &c.prev[i][:1][0] {
			// The row's storage from two frames ago is rebuilt in place, unless a
			// reused string frame left it shared with prev: then parse into
			// fresh storage.
			buf = nil
		}
		if in.Grid != nil {
			if i >= in.GridRows {
				fr.cur[i], fr.curInfo[i] = buf[:0], rowInfo{}
				continue
			}
			row, ok := c.gridRow(in.Grid, i, in.Width, buf)
			if !ok {
				return false
			}
			fr.cur[i], fr.curInfo[i] = row, rowInfo{}
			continue
		}
		if canReuse && i < len(c.prev) && c.prevSrc[i] == line {
			fr.cur[i] = c.prev[i]
			fr.curInfo[i] = c.prevInfo[i]
			fr.same[i] = true
			continue
		}
		if canReuse && !c.noPatch && i < len(c.prev) {
			if idx, pos, ok := patchRow(c.prevSrc[i], line, c.prev[i], c.prevInfo[i], c.patches[i], c.patchPos[i]); ok {
				fr.cur[i] = c.prev[i] // updated in place: prev is replaced by cur below
				c.patches[i], c.patchPos[i] = idx, pos
				fr.curInfo[i] = c.prevInfo[i]
				fr.patched[i] = true
				continue
			}
		}
		if in.LazyFit && i < len(in.Lines) {
			line = in.FitLine(line)
			in.Lines[i] = line
			fr.curSrc[i] = line
		}
		row, ok := c.parseRow(line, buf)
		if !ok {
			if c.wide {
				return false
			}
			// Only this row is unsupported: it is drawn from its line below
			// and the frame is still a cell frame.
			fr.cur[i] = nil
			fr.curInfo[i] = rowInfo{raw: true, reason: c.reason}
			c.reason = ""
			continue
		}
		fr.cur[i] = row
		fr.curInfo[i] = rowInfo{esc: c.rowEsc, simple: c.rowSimple, opq: c.rowOpq}
	}
	return true
}

// deleteGonePlacements deletes the Kitty placements that are gone or moved
// before anything is drawn; a placement that stays is re-sent with its row if
// that is rewritten.
func (c *Cells) deleteGonePlacements(e *cellEmitter) {
	for _, old := range c.placed {
		stays := false
		for _, nw := range c.placedNew {
			if nw == old {
				stays = true
				break
			}
		}
		if !stays {
			e.out = append(e.out, KittyDelete(old.id)...)
		}
	}
}

// drawRow writes what row i needs to change from the previous frame, and
// returns 1 if it wrote anything.
func (c *Cells) drawRow(e *cellEmitter, fr *frameRows, prev [][]cell, havePrev bool, i int) int {
	n := fr.cur[i]
	info := fr.curInfo[i]
	if info.raw {
		if havePrev && i < len(prev) && fr.same[i] {
			return 0 // the same unparsable line as last frame: already on screen
		}
		e.moveRow(i, havePrev && i < len(prev))
		e.moveCol(0)
		e.setPen(0) // the previous row may have left a style on: the line starts from the default
		e.out = append(e.out, ansi.ClearLine...)
		e.out = append(e.out, fr.curSrc[i]...)
		e.out = append(e.out, ansi.Reset...) // the line may leave a style open
		e.pen, e.col = 0, -1
		c.fallbacks = append(c.fallbacks, RowFallback{Row: i, Reason: info.reason})
		return 1
	}
	if !havePrev || i >= len(prev) || c.prevInfo[i].raw {
		e.moveRow(i, havePrev && i < len(prev))
		e.moveCol(0)
		e.setPen(0)
		e.out = append(e.out, 0x1b, '[', 'K')
		e.writeCells(n, 0, len(n))
		e.writeOpaque(info.opq)
		return 1
	}
	o := prev[i]
	if fr.patched[i] {
		c.drawPatched(e, i, n, fr.curSrc[i])
		return 1
	}
	if fr.same[i] || rowsEqual(o, n) && opqEqual(c.prevInfo[i].opq, info.opq) {
		return 0
	}
	diffRow(e, i, o, n)
	if len(info.opq) > 0 {
		e.moveRow(i, true)
		e.writeOpaque(info.opq)
	}
	return 1
}

// drawPatched writes the cells of row i that patchRow changed in place. The
// changed cells are known (same width, one column each), so the spans come from
// them without comparing the whole row. A span runs from its first changed cell
// to its last (merging changes closer than cellGapMerge, like diffRow), and its
// bytes are copied from the new line: they are the span's characters with
// whatever SGR sequences sit between them, which take the terminal through
// exactly the styles the cells carry.
func (c *Cells) drawPatched(e *cellEmitter, i int, n []cell, line string) {
	idx, pos := c.patches[i], c.patchPos[i]
	for a := 0; a < len(idx); {
		first, s, end := a, int(idx[a]), int(idx[a])+1
		a++
		for a < len(idx) && int(idx[a])-end <= cellGapMerge {
			end = int(idx[a]) + 1
			a++
		}
		e.moveRow(i, true)
		e.moveCol(s)
		e.setPen(n[s].st)
		e.out = append(e.out, line[pos[first]:pos[a-1]+1]...)
		e.pen = n[end-1].st
		e.col = end
		if e.width > 0 && end >= e.width {
			e.col = -1 // pending wrap: the cursor position is not reliable
		}
	}
}

// diffRow writes the spans of row i whose cells differ between o (on screen) and
// n, merging differences closer than cellGapMerge, and erases what is left of a
// longer old row.
func diffRow(e *cellEmitter, i int, o, n []cell) {
	ln := len(n)
	lastEnd := 0
	differs := func(k int) bool { return cellAt(o, k) != cellAt(n, k) }
	for s := 0; s < ln; {
		for s < ln && !differs(s) {
			s++
		}
		if s >= ln {
			break
		}
		for s > lastEnd && (isCont(o, s) || isCont(n, s)) {
			s--
		}
		end := s + 1
		for k := s; k < ln; {
			if differs(k) {
				end = k + 1
				k++
				continue
			}
			g := k
			for g < ln && !differs(g) {
				g++
			}
			if g >= ln || g-k > cellGapMerge {
				break
			}
			k = g
		}
		for isCont(o, end) || isCont(n, end) {
			end++
		}
		e.moveRow(i, true)
		e.moveCol(s)
		e.writeCells(n, s, end)
		lastEnd = end
		s = end
	}
	tail := max(ln, lastEnd)
	if len(o) > tail {
		e.moveRow(i, true)
		e.eraseFrom(tail)
	}
}
