package render

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
)

// The default cell renderer (WithCellRenderer, spec #40, decision #4; the line
// renderer is the WithCellRenderer(false) opt-out). Each
// frame's View is parsed once into a grid of cells (a grapheme cluster, its
// column width and an interned style id), diffed against the previous grid
// and only the changed cells are written, using relative cursor movement
// only so the shared lastFrame/liveLines bookkeeping keeps working.
//
// Rows are style-independent in this model: the pen starts as the default at
// the start of every row, so a view whose row leaves a style open does not
// bleed into the next row. Colon-parameter SGR (4:3, 38:2::r:g:b, ...) and
// OSC 8 hyperlinks are part of the style. Anything the grid cannot represent
// (tabs and other control characters, other escapes, unknown SGR codes, a
// view taller than the terminal, width disagreements) makes the frame fall
// back to the line renderer, which is always correct; the reason is kept in
// Cells.reason for the frame log (a slug such as "control_character").

// cell is one terminal column. A wide cluster occupies a head cell (w == 2)
// followed by a continuation cell (w == 0, empty s).
//
// ch is the cell's character: the rune itself, or, with clusterFlag set, an
// index into the renderer's table of multi-rune clusters (a base with
// combining marks, a ZWJ sequence, a flag), so a cell is 8 bytes with no
// pointers: copying a row is a memmove and a stale row keeps no text alive.
type cell struct {
	ch uint32
	w  uint8
	st uint16
}

const clusterFlag = 1 << 31

// maxClusters bounds the cluster table; a frame needing more falls back.
const maxClusters = 4096

var blankCell = cell{ch: ' ', w: 1}

// cellStyle is a decoded SGR state. Colours keep the form the view used
// (basic, bright, 256, RGB) so the terminal gets the same code family.
type cellStyle struct {
	attrs  uint16
	fg, bg cellColor
	ul     uint8  // extended underline style 2-5 (4:2..4:5); 0 for none or plain
	link   uint16 // index into Cells.links; 0 is no link
}

// cellColor: 0 is the default colour; otherwise kind<<24 | value.
type cellColor uint32

const (
	colBasic  = 1 // value 0-7
	colBright = 2 // value 0-7
	col256    = 3 // value 0-255
	colRGB    = 4 // value r<<16|g<<8|b
)

// SGR attribute bits, in the order of their set codes 1,2,3,4,5,7,8,9.
var sgrSetCodes = [8]int{1, 2, 3, 4, 5, 7, 8, 9}

const maxCellStyles = 4096

// cellGapMerge is the widest run of unchanged cells rewritten rather than
// skipped with a cursor move (a CUF costs about four bytes).
const cellGapMerge = 4

// rowInfo describes how a row's source line maps to its cells. A simple row is
// printable ASCII and SGR sequences only, so its cell k sits at byte offset k
// plus the length of every escape sequence before it; esc lists those
// sequences as [start, end) byte offsets, in order. Rows that are not simple
// (tabs, non-ASCII, hyperlinks) have no such mapping and are never patched.
type rowInfo struct {
	esc    []int32
	simple bool
	opq    []opaqueSeg // opaque segments of the row, in order; nil for none
	raw    bool        // the row could not be parsed and is drawn from its line
	reason string      // why, when raw
}

// opaqueSeg is an APC, DCS, PM or SOS string kept verbatim. It occupies no
// column: col is the index of the cell it sits in front of. seq is a
// substring of the view line, so it costs no copy. kid is the kitty image id
// when seq places a kitty graphic (a=T or a=p with i=), else 0.
type opaqueSeg struct {
	col int32
	seq string
	kid uint32
}

// placement is where a kitty image was placed in a frame.
type placement struct {
	id       uint32
	row, col int
}

// RowFallback names a row drawn with the line strategy and why.
type RowFallback struct {
	Row    int    // 0-based row of the frame
	Reason string // slug, for example "control_character"
}

type Cells struct {
	styles    []cellStyle
	ids       map[cellStyle]uint16
	links     []string // raw OSC 8 sequences (with terminator); links[0] unused
	linkIDs   map[string]uint16
	reason    string // why the last frame fell back; "" when it did not
	prev      [][]cell
	cur       [][]cell
	prevSrc   []string // the view lines prev was parsed from, for the row cache
	curSrc    []string
	same      []bool    // rows reused unchanged this frame
	noCache   bool      // tests: parse every row every frame
	noPatch   bool      // tests: never patch a row from the previous frame
	keepBlank bool      // ParseGlyphs: keep trailing default blank cells
	patched   []bool    // rows patched this frame, see patches
	patches   [][]int32 // per row this frame: changed cell indices when the row was patched
	patchPos  [][]int32 // the same changes as byte offsets into the new line
	// prevInfo and curInfo describe the rows of prev and cur for patchRow;
	// rowEsc and rowSimple are what the last parseRow saw.
	prevInfo, curInfo []rowInfo
	rowEsc            []int32
	rowSimple         bool
	rowOpq            []opaqueSeg
	wide              bool        // the last parse failure invalidates the whole frame
	placed, placedNew []placement // kitty placements of the last drawn frame, and scratch
	fallbacks         []RowFallback
	havePrev          bool
	gridPrev          bool          // the previous frame came from a GridSource: its rows have no source lines
	measure           ansi.Measurer // widths are measured the way this Program's terminal draws them
	width             int           // the terminal width the previous frame was fitted to
	clusters          []string      // multi-rune cell text, by index; see clusterFlag
	clusterIDs        map[string]uint32
	out               []byte
	// trans caches the bytes of a pen change, from*transN+to, for the first
	// transN styles: the same few transitions repeat all over a frame, so this
	// replaces building each from its cellStyles. Cleared with the styles.
	trans []string
}

// transN is how many styles the transition cache covers; styles past it
// build their pen change each time.
const transN = 64

func New() *Cells {
	c := &Cells{}
	c.resetStyles()
	return c
}

func (c *Cells) resetStyles() {
	clear(c.trans)
	c.clusters = c.clusters[:0]
	c.clusterIDs = map[string]uint32{}
	c.styles = append(c.styles[:0], cellStyle{})
	c.ids = map[cellStyle]uint16{{}: 0}
	c.links = append(c.links[:0], "")
	c.linkIDs = map[string]uint16{}
}

func (c *Cells) internLink(seq string) (uint16, bool) {
	if id, ok := c.linkIDs[seq]; ok {
		return id, true
	}
	if len(c.links) >= maxCellStyles {
		return 0, false
	}
	id := uint16(len(c.links)) // #nosec G115 -- len(c.links) < maxCellStyles (4096) is checked above, so it fits uint16
	c.links = append(c.links, seq)
	c.linkIDs[seq] = id
	return id, true
}

func (c *Cells) intern(s cellStyle) (uint16, bool) {
	if id, ok := c.ids[s]; ok {
		return id, true
	}
	if len(c.styles) >= maxCellStyles {
		return 0, false
	}
	id := uint16(len(c.styles)) // #nosec G115 -- len(c.styles) < maxCellStyles (4096) is checked above, so it fits uint16
	c.styles = append(c.styles, s)
	c.ids[s] = id
	return id, true
}

// clusterID returns the cell character for multi-rune text, interning it; ok
// is false when the table is full.
func (c *Cells) clusterID(text string) (uint32, bool) {
	if id, ok := c.clusterIDs[text]; ok {
		return id | clusterFlag, true
	}
	if len(c.clusters) >= maxClusters {
		return 0, false
	}
	id := uint32(len(c.clusters)) // #nosec G115 -- len(c.clusters) < maxClusters
	c.clusters = append(c.clusters, text)
	c.clusterIDs[text] = id
	return id | clusterFlag, true
}

// text returns the characters of a cell.
func (c *Cells) text(cl cell) string {
	if cl.ch&clusterFlag != 0 {
		return c.clusters[cl.ch&^clusterFlag]
	}
	if cl.ch == 0 {
		return ""
	}
	return string(rune(cl.ch)) // #nosec G115 -- without clusterFlag, ch is a rune and fits
}

// applySGR applies the parameter string of one SGR sequence (digits, ';' and
// ':' only) to p. It reports false for anything it does not model. The link
// is not part of SGR and survives a reset.
func applySGR(p *cellStyle, params string) bool {
	if params == "" {
		*p = cellStyle{link: p.link}
		return true
	}
	var nums [32]int
	var sub [32]bool // nums[k] follows a ':' (a sub-parameter of nums[k-1])
	n := 0
	v := 0
	colon := false
	for i := 0; i <= len(params); i++ {
		if i == len(params) || params[i] == ';' || params[i] == ':' {
			if n == len(nums) {
				return false
			}
			nums[n], sub[n] = v, colon
			n++
			v = 0
			colon = i < len(params) && params[i] == ':'
			continue
		}
		v = v*10 + int(params[i]-'0')
		if v > 999 {
			return false
		}
	}
	for k := 0; k < n; k++ {
		g := 0 // number of sub-parameters following nums[k]
		for k+1+g < n && sub[k+1+g] {
			g++
		}
		if g > 0 {
			if !applyColonSGR(p, nums[k], nums[k+1:k+1+g]) {
				return false
			}
			k += g
			continue
		}
		switch v := nums[k]; {
		case v == 0:
			*p = cellStyle{link: p.link}
		case v >= 1 && v <= 5 || v == 7 || v == 8 || v == 9:
			for b, code := range sgrSetCodes {
				if code == v {
					p.attrs |= 1 << b
				}
			}
			if v == 4 {
				p.ul = 0
			}
		case v == 22:
			p.attrs &^= 1<<0 | 1<<1
		case v == 23:
			p.attrs &^= 1 << 2
		case v == 24:
			p.attrs &^= 1 << 3
			p.ul = 0
		case v == 25:
			p.attrs &^= 1 << 4
		case v == 27:
			p.attrs &^= 1 << 5
		case v == 28:
			p.attrs &^= 1 << 6
		case v == 29:
			p.attrs &^= 1 << 7
		case v >= 30 && v <= 37:
			p.fg = colBasic<<24 | cellColor(v-30)
		case v >= 90 && v <= 97:
			p.fg = colBright<<24 | cellColor(v-90)
		case v == 39:
			p.fg = 0
		case v >= 40 && v <= 47:
			p.bg = colBasic<<24 | cellColor(v-40)
		case v >= 100 && v <= 107:
			p.bg = colBright<<24 | cellColor(v-100)
		case v == 49:
			p.bg = 0
		case v == 38 || v == 48:
			var col cellColor
			switch {
			case k+2 < n && !sub[k+1] && !sub[k+2] && nums[k+1] == 5 && nums[k+2] <= 255: // #nosec G602 -- k+2 < n <= 32 = len(nums) = len(sub)
				col = col256<<24 | cellColor(nums[k+2]) // #nosec G602 -- k+2 < n <= 32 = len(nums) = len(sub)
				k += 2
			case k+4 < n && !sub[k+1] && !sub[k+2] && !sub[k+3] && !sub[k+4] && // #nosec G602 -- k+4 < n <= 32 = len(nums) = len(sub)
				nums[k+1] == 2 && nums[k+2] <= 255 && nums[k+3] <= 255 && nums[k+4] <= 255: // #nosec G602 -- k+4 < n <= 32 = len(nums) = len(sub)
				col = colRGB<<24 | cellColor(nums[k+2])<<16 | cellColor(nums[k+3])<<8 | cellColor(nums[k+4]) // #nosec G602 -- k+4 < n <= 32 = len(nums) = len(sub)
				k += 4
			default:
				return false
			}
			if v == 38 {
				p.fg = col
			} else {
				p.bg = col
			}
		default:
			return false
		}
	}
	return true
}

// applyColonSGR applies one colon-parameter group head:subs (4:3, 38:5:n,
// 38:2::r:g:b, 38:2:cs:r:g:b) to p. Colours are normalised to the same
// colour the semicolon form gives.
func applyColonSGR(p *cellStyle, head int, subs []int) bool {
	switch head {
	case 4:
		if len(subs) != 1 {
			return false
		}
		switch st := subs[0]; {
		case st == 0:
			p.attrs &^= 1 << 3
			p.ul = 0
		case st == 1:
			p.attrs |= 1 << 3
			p.ul = 0
		case st >= 2 && st <= 5:
			p.attrs |= 1 << 3
			p.ul = uint8(st)
		default:
			return false
		}
		return true
	case 38, 48:
		var col cellColor
		switch {
		case len(subs) == 2 && subs[0] == 5 && subs[1] <= 255:
			col = col256<<24 | cellColor(subs[1]) // #nosec G115 -- subs[1] <= 255 is checked in the case guard
		case len(subs) == 4 && subs[0] == 2 || len(subs) == 5 && subs[0] == 2:
			rgb := subs[len(subs)-3:] // an optional colour-space id sits before r
			if rgb[0] > 255 || rgb[1] > 255 || rgb[2] > 255 {
				return false
			}
			col = colRGB<<24 | cellColor(rgb[0])<<16 | cellColor(rgb[1])<<8 | cellColor(rgb[2]) // #nosec G115 -- rgb components are checked <= 255 just above
		default:
			return false
		}
		if head == 38 {
			p.fg = col
		} else {
			p.bg = col
		}
		return true
	}
	return false
}

// parseRow decodes one view line into cells appended to row[:0]. ok is false
// when the line uses something the grid cannot represent.
func (c *Cells) parseRow(line string, row []cell) ([]cell, bool) {
	row = row[:0]
	var esc []int32
	var opq []opaqueSeg
	simple := true
	if strings.IndexByte(line, '\t') >= 0 {
		line = ansi.ExpandTabs(line)
		simple = false
	}
	var pen cellStyle
	var penID uint16
	head, headStart, headEnd := -1, 0, 0
	headText := "" // the text of the head cluster, when it has grown past one rune
	nonASCII := false
	sawOSC := false

	// setWidth resizes the head cluster's footprint to nw columns.
	setWidth := func(nw int) bool {
		if nw < 1 || nw > 2 {
			return false
		}
		if int(row[head].w) == nw {
			return true
		}
		if nw == 2 {
			row = append(row, cell{st: row[head].st})
		} else {
			row = row[:head+1]
		}
		row[head].w = uint8(nw)
		return true
	}
	// fold appends the rune at line[i:i+size] to the head cluster. foldFail
	// names why it failed, when it did.
	foldFail := "width_disagreement"
	fold := func(i, size int) bool {
		contiguous := headEnd == i
		var text string
		switch {
		case contiguous:
			text = line[headStart : i+size]
			headEnd = i + size
		case headEnd >= 0:
			text = line[headStart:headEnd] + line[i:i+size]
			headEnd = -1
		default:
			text = headText + line[i:i+size]
		}
		headText = text
		id, ok := c.clusterID(text)
		if !ok {
			foldFail = "cluster_table_full"
			return false
		}
		row[head].ch = id
		if contiguous {
			return setWidth(c.measure.Width(text))
		}
		return true
	}

	for i := 0; i < len(line); {
		b := line[i]
		switch {
		case b == 0x1b:
			if i+1 < len(line) && (line[i+1] == '_' || line[i+1] == 'P' || line[i+1] == '^' || line[i+1] == 'X') {
				end, ok := c.parseString(line, i)
				if !ok {
					return nil, false // parseString recorded the reason
				}
				seg := opaqueSeg{col: int32(len(row)), seq: line[i:end]} // #nosec G115 -- a row is far shorter than 2 GiB cells
				seg.kid = kittyPlacementID(seg.seq)
				opq = append(opq, seg)
				sawOSC = true
				simple = false
				i = end
				continue
			}
			if i+1 < len(line) && line[i+1] == ']' {
				end, ok := c.parseLink(line, i, &pen)
				if !ok {
					return nil, false // parseLink recorded the reason
				}
				id, ok := c.intern(pen)
				if !ok {
					c.resetStyles()
					c.havePrev = false
					c.wide = true
					return nil, c.fail("style_table_full")
				}
				penID = id
				sawOSC = true
				simple = false
				i = end
				continue
			}
			if i+1 >= len(line) || line[i+1] != '[' {
				return nil, c.fail("non_CSI_escape")
			}
			j := i + 2
			for j < len(line) && (line[j] >= '0' && line[j] <= '9' || line[j] == ';' || line[j] == ':') {
				j++
			}
			if j >= len(line) || line[j] != 'm' {
				return nil, c.fail("non_SGR_escape")
			}
			if !applySGR(&pen, line[i+2:j]) {
				return nil, c.fail("unsupported_SGR")
			}
			id, ok := c.intern(pen)
			if !ok {
				c.resetStyles()
				c.havePrev = false
				c.wide = true // rows already parsed this frame hold ids the reset invalidated
				return nil, c.fail("style_table_full")
			}
			penID = id
			esc = append(esc, int32(i), int32(j+1)) // #nosec G115 -- a line is far shorter than 2 GiB
			i = j + 1
		case b < 0x20 || b == 0x7f:
			return nil, c.fail("control_character")
		case b < utf8.RuneSelf:
			row = append(row, cell{ch: uint32(b), w: 1, st: penID})
			head, headStart, headEnd = len(row)-1, i, i+1
			i++
		default:
			r, size := utf8.DecodeRuneInString(line[i:])
			if r == utf8.RuneError && size == 1 || r >= 0x80 && r < 0xa0 {
				return nil, c.fail("invalid_UTF_8_or_C1_control")
			}
			nonASCII = true
			simple = false
			w := c.measure.Width(line[i : i+size])
			switch {
			case w == 0:
				if head < 0 || !fold(i, size) {
					return nil, c.fail(foldFail)
				}
			case head >= 0 && headEnd == i && c.measure.Width(line[headStart:i+size]) != int(row[head].w)+w:
				// The rune joins the cluster before it (ZWJ sequence, flag
				// half, prepend...): the pair measures narrower than the sum.
				if !fold(i, size) {
					return nil, c.fail(foldFail)
				}
			default:
				if w > 2 {
					return nil, c.fail("width_disagreement")
				}
				row = append(row, cell{ch: uint32(r), w: uint8(w), st: penID}) // #nosec G115 -- w is at most 2 here, checked just above
				if w == 2 {
					row = append(row, cell{st: penID})
				}
				head, headStart, headEnd = len(row)-1-(w-1), i, i+size
			}
			i += size
		}
	}
	if nonASCII {
		sum := 0
		for _, cl := range row {
			sum += int(cl.w)
		}
		visible := line
		if sawOSC {
			visible = cellStripOSC(line)
		}
		if sum != c.measure.Width(visible) {
			return nil, c.fail("width_disagreement")
		}
	}
	for !c.keepBlank && len(row) > 0 && row[len(row)-1] == blankCell {
		row = row[:len(row)-1]
	}
	c.rowEsc, c.rowSimple, c.rowOpq = esc, simple, opq
	return row, true
}

// fail records why the frame falls back and reports false.
func (c *Cells) fail(reason string) bool {
	c.reason = reason
	return false
}

// parseLink parses the OSC sequence at line[i:] (line[i:i+2] is ESC ]), which
// must be an OSC 8 hyperlink open or close terminated by BEL or ST, and sets
// pen.link. It returns the index after the sequence.
func (c *Cells) parseLink(line string, i int, pen *cellStyle) (int, bool) {
	end, tl := -1, 0
	for j := i + 2; j < len(line); j++ {
		if line[j] == 0x07 {
			end, tl = j, 1
			break
		}
		if line[j] == 0x1b && j+1 < len(line) && line[j+1] == '\\' {
			end, tl = j, 2
			break
		}
	}
	if end < 0 {
		return 0, c.fail("unterminated_OSC")
	}
	body := line[i+2 : end]
	if len(body) < 2 || body[0] != '8' || body[1] != ';' {
		return 0, c.fail("non_hyperlink_OSC")
	}
	for k := 0; k < len(body); k++ {
		if body[k] < 0x20 && body[k] != 0 || body[k] == 0x7f {
			return 0, c.fail("control_character")
		}
	}
	rest := body[2:] // params;uri
	sep := strings.IndexByte(rest, ';')
	if sep < 0 {
		return 0, c.fail("malformed_hyperlink")
	}
	if rest[sep+1:] == "" {
		pen.link = 0
	} else {
		id, ok := c.internLink(line[i : end+tl])
		if !ok {
			c.resetStyles()
			c.havePrev = false
			c.wide = true // rows already parsed this frame hold ids the reset invalidated
			return 0, c.fail("style_table_full")
		}
		pen.link = id
	}
	return end + tl, true
}

// tmuxDCS opens a tmux passthrough string.
const tmuxDCS = "\x1bPtmux;"

// parseString parses the APC, DCS, PM or SOS string at line[i:] (ESC then one
// of _ P ^ X), which ends at ST (ESC \), and returns the index after it.
//
// A "DCS tmux;" string, which wraps another sequence with its ESC bytes
// doubled, ends at the first ESC \ that is not part of such a pair.
func (c *Cells) parseString(line string, i int) (int, bool) {
	if strings.HasPrefix(line[i:], tmuxDCS) {
		for j := i + len(tmuxDCS); j < len(line); j++ {
			if line[j] != 0x1b {
				continue
			}
			switch {
			case j+1 < len(line) && line[j+1] == 0x1b:
				j++ // a doubled ESC belongs to the wrapped sequence
			case j+1 < len(line) && line[j+1] == '\\':
				return j + 2, true
			default:
				return 0, c.fail("malformed_string_sequence")
			}
		}
		return 0, c.fail("unterminated_string_sequence")
	}
	for j := i + 2; j < len(line); j++ {
		if line[j] != 0x1b {
			continue
		}
		if j+1 < len(line) && line[j+1] == '\\' {
			return j + 2, true
		}
		return 0, c.fail("malformed_string_sequence")
	}
	return 0, c.fail("unterminated_string_sequence")
}

// kittyPlacementID returns the image id of a kitty graphics command that
// places an image (action T or p, with an i= key), or 0.
func kittyPlacementID(seq string) uint32 {
	if inner, ok := strings.CutPrefix(seq, tmuxDCS); ok {
		seq = strings.ReplaceAll(strings.TrimSuffix(inner, "\x1b\\"), "\x1b\x1b", "\x1b")
	}
	if len(seq) < 4 || seq[1] != '_' || seq[2] != 'G' {
		return 0
	}
	ctl := seq[3:]
	if k := strings.IndexAny(ctl, ";\x1b"); k >= 0 {
		ctl = ctl[:k]
	}
	var id uint64
	act := byte('t')
	for ctl != "" {
		var kv string
		kv, ctl, _ = strings.Cut(ctl, ",")
		switch {
		case strings.HasPrefix(kv, "a=") && len(kv) == 3:
			act = kv[2]
		case strings.HasPrefix(kv, "i="):
			v, err := strconv.ParseUint(kv[2:], 10, 32)
			if err != nil {
				return 0
			}
			id = v
		}
	}
	if act != 'T' && act != 'p' {
		return 0
	}
	return uint32(id) // #nosec G115 -- ParseUint with bitSize 32
}

// cellStripOSC removes every OSC (BEL or ST terminated) and APC, DCS, PM or
// SOS (ST terminated) sequence from s.
func cellStripOSC(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && (s[i+1] == ']' || s[i+1] == '_' || s[i+1] == 'P' || s[i+1] == '^' || s[i+1] == 'X') {
			bel := s[i+1] == ']'
			j := i + 2
			for j < len(s) && !(bel && s[j] == 0x07) && !(s[j] == 0x1b && j+1 < len(s) && s[j+1] == '\\') {
				j++
			}
			if j < len(s) {
				if s[j] == 0x07 {
					j++
				} else {
					j += 2
				}
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func cellAt(row []cell, i int) cell {
	if i < len(row) {
		return row[i]
	}
	return blankCell
}

func isCont(row []cell, i int) bool { return i < len(row) && row[i].w == 0 }

func opqEqual(a, b []opaqueSeg) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].col != b[i].col || a[i].seq != b[i].seq {
			return false
		}
	}
	return true
}

func rowsEqual(a, b []cell) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Reason is why the last Frame fell back to the line renderer, a slug such as
// "control_character"; "" when it did not.
func (c *Cells) Reason() string { return c.reason }

// Valid reports whether a previous frame is held for diffing.
func (c *Cells) Valid() bool { return c.havePrev }

// Invalidate drops the previous frame, so the next Frame redraws every row.
func (c *Cells) Invalidate() { c.havePrev = false }

// Width is the terminal width the previous frame was fitted to.
func (c *Cells) Width() int { return c.width }

// SetNoCache makes every row be parsed every frame. For tests and benchmarks.
func (c *Cells) SetNoCache(v bool) { c.noCache = v }

// SetNoPatch stops rows being patched from the previous frame. For tests.
func (c *Cells) SetNoPatch(v bool) { c.noPatch = v }

// PatchedRows reports, per row of the last frame, whether it was patched
// rather than parsed. For tests.
func (c *Cells) PatchedRows() []bool { return c.patched }

// GridRows is the number of rows in the held previous frame. For tests.
func (c *Cells) GridRows() int { return len(c.prev) }
