package render

import "strings"

// GridSource is a grid of cells that Cells.Frame draws without parsing a
// string: the entry point for models that draw cells directly.
type GridSource interface {
	// Cell returns the cell at column x of row y: its cluster text ("" for the
	// continuation of a wide cluster), its width (0, 1 or 2) and its style.
	Cell(x, y int) (text string, width uint8, st Style)
}

// gridMemo remembers the style gridRow resolved last and the id it got, so a
// run of cells of one style costs one map lookup, not one each. resetStyles
// clears it, since the ids it holds are gone.
type gridMemo struct {
	gridLast Style
	gridSid  uint16
	gridHave bool
}

// gridRow converts row y of g (width columns) to cells, interning styles and
// clusters in c. Trailing default blanks are dropped, like parseRow. ok is
// false, with c.reason set, when a cell cannot be represented.
func (c *Cells) gridRow(g GridSource, y, width int, row []cell) ([]cell, bool) {
	row = row[:0]
	for x := 0; x < width; x++ {
		text, w, st := g.Cell(x, y)
		var sid uint16
		var ok bool
		if c.gridHave && st == c.gridLast {
			sid, ok = c.gridSid, true // the same style as the cell before
		} else {
			cs := importStyle(st)
			if st.Link != "" {
				id, ok := c.internLink("\x1b]8;;" + st.Link + "\x1b\\")
				if !ok {
					c.resetStyles()
					c.havePrev = false
					return nil, c.fail("style_table_full")
				}
				cs.link = id
			}
			sid, ok = c.intern(cs)
			if !ok {
				c.resetStyles()
				c.havePrev = false
				return nil, c.fail("style_table_full")
			}
			c.gridLast, c.gridSid, c.gridHave = st, sid, true
		}
		if w == 0 {
			row = append(row, cell{st: sid})
			continue
		}
		if w == 1 && len(text) == 1 && text[0] >= 0x20 && text[0] < 0x7f {
			// A printable ASCII cell, the common case: no scan, no rune slice.
			row = append(row, cell{ch: uint32(text[0]), w: 1, st: sid})
			continue
		}
		if w > 2 || text == "" || strings.ContainsFunc(text, func(r rune) bool { return r < 0x20 || r == 0x7f || r >= 0x80 && r < 0xa0 }) {
			return nil, c.fail("control_character")
		}
		var ch uint32
		if r := []rune(text); len(r) == 1 {
			ch = uint32(r[0]) // #nosec G115 -- a rune is non-negative and fits
		} else if ch, ok = c.clusterID(text); !ok {
			return nil, c.fail("cluster_table_full")
		}
		row = append(row, cell{ch: ch, w: w, st: sid})
	}
	for len(row) > 0 && row[len(row)-1] == blankCell {
		row = row[:len(row)-1]
	}
	return row, true
}
