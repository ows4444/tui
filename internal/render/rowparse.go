package render

import (
	"strings"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
)

// parseRow decodes one view line into cells appended to row[:0]. ok is false
// when the line uses something the grid cannot represent.
func (c *Cells) parseRow(line string, row []cell) ([]cell, bool) {
	p := rowParser{c: c, line: line, row: row[:0], simple: true, head: -1, foldFail: "width_disagreement"}
	if strings.IndexByte(line, '\t') >= 0 {
		p.line = ansi.ExpandTabs(line)
		p.simple = false
	}
	if n := strings.Count(p.line, "\x1b"); n > 0 {
		p.esc = make([]int32, 0, 2*n) // one allocation, not one per doubling
	}
	for i := 0; i < len(p.line); {
		var ok bool
		if b := p.line[i]; b == 0x1b {
			i, ok = p.escape(i)
		} else {
			i, ok = p.char(i, b)
		}
		if !ok {
			return nil, false
		}
	}
	return p.finish()
}

// rowParser is the state of parseRow: the cells so far, the pen the escape
// sequences have set, and the head cluster that combining runes fold into.
type rowParser struct {
	c      *Cells
	line   string
	row    []cell
	esc    []int32
	opq    []opaqueSeg
	simple bool
	pen    cellStyle
	penID  uint16
	// head is the index of the cell that starts the last cluster, and
	// headStart:headEnd is where its text is in line (headEnd is -1 once the
	// cluster has grown past what line holds contiguously).
	head, headStart, headEnd int
	headText                 string // the text of the head cluster, when it has grown past one rune
	nonASCII, sawOSC         bool
	foldFail                 string // why fold failed, when it did
}

// setWidth resizes the head cluster's footprint to nw columns.
func (p *rowParser) setWidth(nw int) bool {
	if nw < 1 || nw > 2 {
		return false
	}
	if int(p.row[p.head].w) == nw {
		return true
	}
	if nw == 2 {
		p.row = append(p.row, cell{st: p.row[p.head].st})
	} else {
		p.row = p.row[:p.head+1]
	}
	p.row[p.head].w = uint8(nw)
	return true
}

// fold appends the rune at line[i:i+size] to the head cluster.
func (p *rowParser) fold(i, size int) bool {
	contiguous := p.headEnd == i
	var text string
	switch {
	case contiguous:
		text = p.line[p.headStart : i+size]
		p.headEnd = i + size
	case p.headEnd >= 0:
		text = p.line[p.headStart:p.headEnd] + p.line[i:i+size]
		p.headEnd = -1
	default:
		text = p.headText + p.line[i:i+size]
	}
	p.headText = text
	id, ok := p.c.clusterID(text)
	if !ok {
		p.foldFail = "cluster_table_full"
		return false
	}
	p.row[p.head].ch = id
	if contiguous {
		return p.setWidth(p.c.measure.Width(text))
	}
	return true
}

// styleFull handles a full style table: the rows already parsed this frame hold
// ids the reset invalidated, so the whole frame falls back.
func (p *rowParser) styleFull() bool {
	p.c.resetStyles()
	p.c.havePrev = false
	p.c.wide = true
	return p.c.fail("style_table_full")
}

// escape handles the escape sequence at line[i]: an opaque string (a Kitty
// graphic, a sixel), an inline image, a hyperlink, or an SGR. It returns where
// the line goes on.
func (p *rowParser) escape(i int) (int, bool) {
	c, line := p.c, p.line
	if i+1 < len(line) && (line[i+1] == '_' || line[i+1] == 'P' || line[i+1] == '^' || line[i+1] == 'X') {
		end, ok := c.parseString(line, i)
		if !ok {
			return i, false // parseString recorded the reason
		}
		seg := opaqueSeg{col: int32(len(p.row)), seq: line[i:end]} // #nosec G115 -- a row is far shorter than 2 GiB cells
		seg.kid = kittyPlacementID(seg.seq)
		p.opq = append(p.opq, seg)
		p.sawOSC, p.simple = true, false
		return end, true
	}
	if strings.HasPrefix(line[i:], inlineImageOSC) {
		end, ok := c.parseInlineImage(line, i)
		if !ok {
			return i, false // parseInlineImage recorded the reason
		}
		p.opq = append(p.opq, opaqueSeg{col: int32(len(p.row)), seq: line[i:end]}) // #nosec G115 -- a row is far shorter than 2 GiB cells
		p.sawOSC, p.simple = true, false
		return end, true
	}
	if i+1 < len(line) && line[i+1] == ']' {
		end, ok := c.parseLink(line, i, &p.pen)
		if !ok {
			return i, false // parseLink recorded the reason
		}
		id, ok := c.intern(p.pen)
		if !ok {
			return i, p.styleFull()
		}
		p.penID = id
		p.sawOSC, p.simple = true, false
		return end, true
	}
	if i+1 >= len(line) || line[i+1] != '[' {
		return i, c.fail("non_CSI_escape")
	}
	j := i + 2
	for j < len(line) && (line[j] >= '0' && line[j] <= '9' || line[j] == ';' || line[j] == ':') {
		j++
	}
	if j >= len(line) || line[j] != 'm' {
		return i, c.fail("non_SGR_escape")
	}
	if !applySGR(&p.pen, line[i+2:j]) {
		return i, c.fail("unsupported_SGR")
	}
	id, ok := c.intern(p.pen)
	if !ok {
		return i, p.styleFull()
	}
	p.penID = id
	p.esc = append(p.esc, int32(i), int32(j+1)) // #nosec G115 -- a line is far shorter than 2 GiB
	return j + 1, true
}

// char handles the printable character that starts at line[i] (b is its first
// byte), appending its cell or folding it into the head cluster.
func (p *rowParser) char(i int, b byte) (int, bool) {
	c, line := p.c, p.line
	switch {
	case b < 0x20 || b == 0x7f:
		return i, c.fail("control_character")
	case b < utf8.RuneSelf:
		p.row = append(p.row, cell{ch: uint32(b), w: 1, st: p.penID})
		p.head, p.headStart, p.headEnd = len(p.row)-1, i, i+1
		return i + 1, true
	}
	r, size := utf8.DecodeRuneInString(line[i:])
	if r == utf8.RuneError && size == 1 || r >= 0x80 && r < 0xa0 {
		return i, c.fail("invalid_UTF_8_or_C1_control")
	}
	p.nonASCII, p.simple = true, false
	w := c.measure.Width(line[i : i+size])
	switch {
	case w == 0:
		if p.head < 0 || !p.fold(i, size) {
			return i, c.fail(p.foldFail)
		}
	case p.head >= 0 && p.headEnd == i && c.measure.Width(line[p.headStart:i+size]) != int(p.row[p.head].w)+w:
		// The rune joins the cluster before it (ZWJ sequence, flag
		// half, prepend...): the pair measures narrower than the sum.
		if !p.fold(i, size) {
			return i, c.fail(p.foldFail)
		}
	default:
		if w > 2 {
			return i, c.fail("width_disagreement")
		}
		p.row = append(p.row, cell{ch: uint32(r), w: uint8(w), st: p.penID}) // #nosec G115 -- w is at most 2 here, checked just above
		if w == 2 {
			p.row = append(p.row, cell{st: p.penID})
		}
		p.head, p.headStart, p.headEnd = len(p.row)-1-(w-1), i, i+size
	}
	return i + size, true
}

// finish checks the widths the parse added up to and trims the blank tail.
func (p *rowParser) finish() ([]cell, bool) {
	c, row := p.c, p.row
	if p.nonASCII {
		sum := 0
		for _, cl := range row {
			sum += int(cl.w)
		}
		visible := p.line
		if p.sawOSC {
			visible = cellStripOSC(p.line)
		}
		if sum != c.measure.Width(visible) {
			return nil, c.fail("width_disagreement")
		}
	}
	for !c.keepBlank && len(row) > 0 && row[len(row)-1] == blankCell {
		row = row[:len(row)-1]
	}
	c.rowEsc, c.rowSimple, c.rowOpq = p.esc, p.simple, p.opq
	return row, true
}
