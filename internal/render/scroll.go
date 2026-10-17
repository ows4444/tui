package render

import "strconv"

// Hardware scrolling. A log or a list that scrolls by a line changes every row
// it shows, and diffing row by row writes them all again. When a block of rows
// is the previous frame's block moved up or down by a few lines, the terminal can
// do that move itself (DECSTBM to set the rows, then SU or SD), and only the
// rows that are new have to be written. The check is made on the view's lines,
// before anything is parsed, and the previous frame's rows are moved the same
// way so every later step sees what is now on the screen.

const (
	// scrollMaxShift is the most lines a block may move and still be found.
	scrollMaxShift = 16
	// scrollMinBenefit is how many rows must stop needing a rewrite before a
	// scroll is worth its bytes.
	scrollMinBenefit = 4
)

// scrollMove is a block of rows that moved: rows [top, bottom) of the screen
// scroll up by n lines, or down by -n.
type scrollMove struct{ top, bottom, n int }

// appendSeq appends to b the sequences that make the move on the terminal: set
// the scroll region (which homes the cursor), scroll it, and put the region back.
func (m scrollMove) appendSeq(b []byte) []byte {
	b = append(b, 0x1b, '[')
	b = strconv.AppendInt(b, int64(m.top+1), 10)
	b = append(b, ';')
	b = strconv.AppendInt(b, int64(m.bottom), 10)
	b = append(b, 'r', 0x1b, '[')
	if m.n > 0 {
		b = strconv.AppendInt(b, int64(m.n), 10)
		b = append(b, 'S')
	} else {
		b = strconv.AppendInt(b, int64(-m.n), 10)
		b = append(b, 'T')
	}
	return append(b, 0x1b, '[', 'r')
}

// detectScroll looks for a block of rows of in.Lines that is a block of the
// previous frame's rows moved by a few lines, and reports the move that saves
// the most rewriting. It makes no change and allocates nothing.
func (c *Cells) detectScroll(in Frame) (scrollMove, bool) {
	rows := in.Max
	if !in.HardwareScroll || in.Grid != nil || c.noCache || !c.havePrev || c.gridPrev ||
		in.Height <= 0 || rows < 2*scrollMinBenefit || rows > in.Height ||
		in.PrevRows != rows || len(c.prev) != rows || len(c.prevSrc) != rows ||
		c.width != in.Width || in.Width <= 0 {
		return scrollMove{}, false
	}
	changed := 0
	for i := 0; i < rows; i++ {
		if lineAt(in.Lines, i) != c.prevSrc[i] {
			changed++
		}
	}
	if changed < scrollMinBenefit {
		return scrollMove{}, false
	}
	var best scrollMove
	bestBenefit := 0
	for s := 1; s <= min(scrollMaxShift, rows/2); s++ {
		for dir := 1; dir >= -1; dir -= 2 {
			// Row i of the new frame is row i+s (scrolled up) or i-s (down) of the old one.
			run, benefit, start := 0, 0, 0
			for i := 0; i <= rows; i++ {
				if i < rows {
					if j := i + dir*s; j >= 0 && j < rows && lineAt(in.Lines, i) == c.prevSrc[j] {
						if run == 0 {
							start, benefit = i, 0
						}
						run++
						if lineAt(in.Lines, i) != c.prevSrc[i] {
							benefit++
						}
						continue
					}
				}
				// The run [start, i) has ended.
				if run > 0 && benefit > bestBenefit && benefit >= scrollMinBenefit && benefit*2 >= changed && c.scrollable(start, i, s, dir) {
					bestBenefit = benefit
					if dir > 0 {
						best = scrollMove{top: start, bottom: i + s, n: s}
					} else {
						best = scrollMove{top: start - s, bottom: i, n: -s}
					}
				}
				run, benefit = 0, 0
			}
		}
	}
	return best, bestBenefit > 0
}

// lineAt is lines[i], or "" past the end: a missing row is blank.
func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}

// scrollable reports whether the rows a move of the run [start, end) by s would
// touch hold only plain text: a row with an image, a link target or a line that
// could not be parsed is left to the ordinary diff.
func (c *Cells) scrollable(start, end, s, dir int) bool {
	lo, hi := start, end+s
	if dir < 0 {
		lo, hi = start-s, end
	}
	if lo < 0 || hi > len(c.prevInfo) {
		return false
	}
	for i := lo; i < hi; i++ {
		if c.prevInfo[i].raw || len(c.prevInfo[i].opq) > 0 {
			return false
		}
	}
	return true
}

// applyScroll moves the previous frame's rows as the terminal will move them:
// the rows that scroll off are dropped and the rows that open up are blank. The
// frame-before-last's cell storage (cur) moves with them, so a row of storage
// that shares its cells with a row of prev still sits at the same index and the
// check in buildRows that keeps it from being parsed into still sees it. The
// storage of the rows that scroll off serves the blank rows, so nothing is
// allocated.
func (c *Cells) applyScroll(m scrollMove) {
	for len(c.cur) < m.bottom { // the spare storage grows with the frames: it may be shorter than the move
		c.cur = append(c.cur, nil)
	}
	var offP, offC [scrollMaxShift][]cell
	if m.n > 0 {
		k := m.n
		for j := 0; j < k; j++ {
			offP[j], offC[j] = c.prev[m.top+j], c.cur[m.top+j]
		}
		for i := m.top; i < m.bottom-k; i++ {
			c.prev[i], c.prevSrc[i], c.prevInfo[i], c.cur[i] = c.prev[i+k], c.prevSrc[i+k], c.prevInfo[i+k], c.cur[i+k]
		}
		for j := 0; j < k; j++ {
			r := m.bottom - k + j
			c.prev[r], c.prevSrc[r], c.prevInfo[r], c.cur[r] = offP[j][:0], "", rowInfo{}, offC[j][:0]
		}
		return
	}
	k := -m.n
	for j := 0; j < k; j++ {
		offP[j], offC[j] = c.prev[m.bottom-k+j], c.cur[m.bottom-k+j]
	}
	for i := m.bottom - 1; i >= m.top+k; i-- {
		c.prev[i], c.prevSrc[i], c.prevInfo[i], c.cur[i] = c.prev[i-k], c.prevSrc[i-k], c.prevInfo[i-k], c.cur[i-k]
	}
	for j := 0; j < k; j++ {
		r := m.top + j
		c.prev[r], c.prevSrc[r], c.prevInfo[r], c.cur[r] = offP[j][:0], "", rowInfo{}, offC[j][:0]
	}
}
