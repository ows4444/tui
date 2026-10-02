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
	newLines, max, lazyFit := in.Lines, in.Max, in.LazyFit
	full := in.PrevRows == 0
	c.reason = ""
	c.wide = false
	c.fallbacks = c.fallbacks[:0]
	if in.Measurer != c.measure {
		// Cell widths and cached rows were computed under the old measurer.
		c.measure = in.Measurer
		c.havePrev = false
	}
	if in.Height > 0 && max > in.Height {
		c.havePrev = false
		c.reason = "view_taller_than_terminal"
		return "", Stats{}, false
	}
	for len(c.cur) < max {
		c.cur = append(c.cur, nil)
	}
	cur := c.cur[:max]
	if cap(c.curSrc) < max {
		c.curSrc = make([]string, max)
	}
	curSrc := c.curSrc[:max]
	// A row whose source line is unchanged since the previous frame keeps its
	// parsed cells (shared with prev, never mutated) instead of being parsed.
	if cap(c.same) < max {
		c.same = make([]bool, max)
	}
	same := c.same[:max]
	for len(c.patches) < max {
		c.patches = append(c.patches, nil)
		c.patchPos = append(c.patchPos, nil)
	}
	for len(c.curInfo) < max {
		c.curInfo = append(c.curInfo, rowInfo{})
	}
	curInfo := c.curInfo[:max]
	if cap(c.patched) < max {
		c.patched = make([]bool, max)
	}
	patched := c.patched[:max]
	canReuse := !c.noCache && c.havePrev && len(c.prev) == len(c.prevSrc) && !c.gridPrev && in.Grid == nil
	for i := 0; i < max; i++ {
		var line string
		if i < len(newLines) {
			line = newLines[i]
		}
		curSrc[i] = line
		same[i] = false
		patched[i] = false
		if in.Grid != nil {
			// The row's storage from two frames ago is rebuilt in place, unless a
			// reused string frame left it shared with prev.
			buf := cur[i]
			if i < len(c.prev) && cap(buf) > 0 && cap(c.prev[i]) > 0 && &buf[:1][0] == &c.prev[i][:1][0] {
				buf = nil
			}
			if i >= in.GridRows {
				cur[i], curInfo[i] = buf[:0], rowInfo{}
				continue
			}
			row, ok := c.gridRow(in.Grid, i, in.Width, buf)
			if !ok {
				c.havePrev = false
				return "", Stats{}, false
			}
			cur[i], curInfo[i] = row, rowInfo{}
			continue
		}
		if canReuse && i < len(c.prev) && c.prevSrc[i] == line {
			cur[i] = c.prev[i]
			curInfo[i] = c.prevInfo[i]
			same[i] = true
			continue
		}
		buf := cur[i]
		if i < len(c.prev) && cap(buf) > 0 && cap(c.prev[i]) > 0 && &buf[:1][0] == &c.prev[i][:1][0] {
			buf = nil // shared with prev from a reused frame: parse into fresh storage
		}
		if canReuse && !c.noPatch && i < len(c.prev) {
			if idx, pos, ok := patchRow(c.prevSrc[i], line, c.prev[i], c.prevInfo[i], c.patches[i], c.patchPos[i]); ok {
				cur[i] = c.prev[i] // updated in place: prev is replaced by cur below
				c.patches[i], c.patchPos[i] = idx, pos
				curInfo[i] = c.prevInfo[i]
				patched[i] = true
				continue
			}
		}
		if lazyFit && i < len(newLines) {
			line = in.FitLine(line)
			newLines[i] = line
			curSrc[i] = line
		}
		row, ok := c.parseRow(line, buf)
		if !ok {
			if c.wide {
				c.havePrev = false
				return "", Stats{}, false
			}
			// Only this row is unsupported: it is drawn from its line below
			// and the frame is still a cell frame.
			cur[i] = nil
			curInfo[i] = rowInfo{raw: true, reason: c.reason}
			c.reason = ""
			continue
		}
		cur[i] = row
		curInfo[i] = rowInfo{esc: c.rowEsc, simple: c.rowSimple, opq: c.rowOpq}
	}
	c.placedNew = c.placedNew[:0]
	for i := 0; i < max; i++ {
		for _, sg := range curInfo[i].opq {
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
	changed := 0
	// Kitty placements that are gone or moved are deleted before anything is
	// drawn; a placement that stays is re-sent with its row if that is rewritten.
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

	for i := 0; i < max; i++ {
		n := cur[i]
		if curInfo[i].raw {
			if havePrev && i < len(prev) && same[i] {
				continue // the same unparsable line as last frame: already on screen
			}
			e.moveRow(i, havePrev && i < len(prev))
			e.moveCol(0)
			e.setPen(0) // the previous row may have left a style on: the line starts from the default
			e.out = append(e.out, ansi.ClearLine...)
			e.out = append(e.out, curSrc[i]...)
			e.out = append(e.out, ansi.Reset...) // the line may leave a style open
			e.pen, e.col = 0, -1
			changed++
			c.fallbacks = append(c.fallbacks, RowFallback{Row: i, Reason: curInfo[i].reason})
			continue
		}
		if !havePrev || i >= len(prev) || c.prevInfo[i].raw {
			e.moveRow(i, havePrev && i < len(prev))
			e.moveCol(0)
			e.setPen(0)
			e.out = append(e.out, 0x1b, '[', 'K')
			e.writeCells(n, 0, len(n))
			e.writeOpaque(curInfo[i].opq)
			changed++
			continue
		}
		o := prev[i]
		if patched[i] {
			// The changed cells are known (same width, one column each), so
			// the spans come from them without comparing the whole row. A span
			// runs from its first changed cell to its last (merging changes
			// closer than cellGapMerge, like the scan below), and its bytes
			// are copied from the new line: they are the span's characters
			// with whatever SGR sequences sit between them, which take the
			// terminal through exactly the styles the cells carry.
			changed++
			idx, pos := c.patches[i], c.patchPos[i]
			line := curSrc[i]
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
			continue
		}
		if same[i] || rowsEqual(o, n) && opqEqual(c.prevInfo[i].opq, curInfo[i].opq) {
			continue
		}
		changed++
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
		tail := ln
		if lastEnd > tail {
			tail = lastEnd
		}
		if len(o) > tail {
			e.moveRow(i, true)
			e.eraseFrom(tail)
		}
		if len(curInfo[i].opq) > 0 {
			e.moveRow(i, true)
			e.writeOpaque(curInfo[i].opq)
		}
	}
	e.moveRow(max-1, true)
	e.setPen(0)

	c.out = e.out
	c.placed, c.placedNew = c.placedNew, c.placed
	c.width = in.Width
	c.prev, c.cur = cur, prev
	c.prevInfo, c.curInfo = curInfo, c.prevInfo
	c.prevSrc, c.curSrc = curSrc, c.prevSrc
	c.havePrev = true
	c.gridPrev = in.Grid != nil
	return string(e.out), Stats{Rows: max, Changed: changed, Full: full, Fallbacks: c.fallbacks}, true
}
