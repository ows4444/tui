package cellbuf

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/render"
)

// Cell is one terminal column. A wide cluster occupies a head cell (Width 2)
// followed by a continuation cell (Width 0, empty Cluster). A blank cell is a
// single space of Width 1.
type Cell struct {
	// Cluster is the cell's grapheme cluster, "" for a continuation cell.
	Cluster string
	// Width is the columns the cluster covers: 1 or 2 for a head, 0 for a
	// continuation.
	Width uint8
	// Style is the cell's style in the owning Buffer's table.
	Style StyleID
}

// Rect is a rectangle of cells; X and Y are the top-left corner.
type Rect struct {
	X, Y, W, H int
}

// ErrUnsupported is wrapped by the errors Parse and SetStyled return for a
// string the cell grid cannot represent.
var ErrUnsupported = errors.New("cellbuf: unsupported string")

const maxStyles = 1<<16 - 1

// store is the storage shared by a Buffer and all its Sub views.
type store struct {
	cells  []Cell
	stride int
	styles []Style
	ids    map[Style]StyleID
	m      ansi.Measurer
}

// Buffer is a grid of cells, or a clipped view (see Sub) of one.
type Buffer struct {
	s *store
	r Rect // the visible region in store coordinates
}

func blank(id StyleID) Cell { return Cell{Cluster: " ", Width: 1, Style: id} }

// New returns a width x height Buffer of blank default-style cells. Negative
// sizes are taken as 0.
func New(width, height int) *Buffer {
	width, height = max(width, 0), max(height, 0)
	s := &store{
		cells:  make([]Cell, width*height),
		stride: width,
		styles: []Style{{}},
		ids:    map[Style]StyleID{{}: 0},
	}
	for i := range s.cells {
		s.cells[i] = blank(0)
	}
	return &Buffer{s: s, r: Rect{0, 0, width, height}}
}

// SetMeasurer sets how cluster widths are measured by SetString, SetStyled,
// Fill and ParseMeasured, for every view of the same cells. The zero Measurer
// follows the process-wide setting of package ansi. Cells already in the
// Buffer are not re-measured.
func (b *Buffer) SetMeasurer(m ansi.Measurer) { b.s.m = m }

// Width returns the number of columns of b.
func (b *Buffer) Width() int { return b.r.W }

// Height returns the number of rows of b.
func (b *Buffer) Height() int { return b.r.H }

// Bounds returns b's own rectangle, {0, 0, Width, Height}.
func (b *Buffer) Bounds() Rect { return Rect{0, 0, b.r.W, b.r.H} }

// StyleID adds st to the style table, if it is not there, and returns its id.
// Control characters are removed from st.Link. When the table is full (65535
// styles) it returns 0, the default style.
func (b *Buffer) StyleID(st Style) StyleID {
	s := b.s
	if st.Link != "" {
		st.Link = strings.Map(func(r rune) rune {
			if r < 0x20 || r == 0x7f || r >= 0x80 && r < 0xa0 {
				return -1
			}
			return r
		}, st.Link)
	}
	if id, ok := s.ids[st]; ok {
		return id
	}
	if len(s.styles) >= maxStyles {
		return 0
	}
	id := StyleID(len(s.styles)) // #nosec G115 -- bounded by maxStyles just above
	s.styles = append(s.styles, st)
	s.ids[st] = id
	return id
}

// Style returns the style an id names; the zero Style for an unknown id.
func (b *Buffer) Style(id StyleID) Style {
	if int(id) < len(b.s.styles) {
		return b.s.styles[id]
	}
	return Style{}
}

// At returns the cell at column x, row y of b; a blank default cell when the
// position is outside b.
func (b *Buffer) At(x, y int) Cell {
	if x < 0 || y < 0 || x >= b.r.W || y >= b.r.H {
		return blank(0)
	}
	return b.s.cells[(b.r.Y+y)*b.s.stride+b.r.X+x]
}

// Sub returns a view of the part of r (relative to b) that lies inside b. The
// view shares cells and styles with b, so drawing into it changes b, and its
// own coordinates start at r's corner. Drawing is clipped to the view. When a
// write lands on one half of a wide cluster that straddles the view's edge, the
// other half, outside the view, is blanked too so no half cluster remains.
func (b *Buffer) Sub(r Rect) *Buffer { return &Buffer{s: b.s, r: b.subRect(r)} }

// subRect is the part of r (relative to b) inside b, in store coordinates.
func (b *Buffer) subRect(r Rect) Rect {
	x0, y0 := max(r.X, 0), max(r.Y, 0)
	w, h := min(r.X+r.W, b.r.W)-x0, min(r.Y+r.H, b.r.H)-y0
	if w <= 0 || h <= 0 {
		return Rect{X: b.r.X + x0, Y: b.r.Y + y0}
	}
	return Rect{X: b.r.X + x0, Y: b.r.Y + y0, W: w, H: h}
}

// Clear blanks every cell of b with the default style.
func (b *Buffer) Clear() { b.Fill(b.Bounds(), " ", 0) }

// item is one cluster to place.
type item struct {
	text string
	w    int
	id   StyleID
}

// breakAt blanks the other half of a wide cluster that the cell at (col, row)
// belongs to, in store coordinates, before that cell is overwritten.
func (s *store) breakAt(col, row int) {
	i := row*s.stride + col
	switch c := s.cells[i]; {
	case c.Width == 0 && col > 0:
		s.cells[i-1] = blank(s.cells[i-1].Style)
	case c.Width == 2 && col+1 < s.stride:
		s.cells[i+1] = blank(c.Style)
	}
}

func (s *store) put(col, row int, c Cell) {
	s.breakAt(col, row)
	s.cells[row*s.stride+col] = c
}

// place writes items on row y starting at column x of b, clipped to b. A wide
// cluster that is not wholly inside b (at either edge) is replaced by a blank
// in its visible column. It returns the columns of b written.
func (b *Buffer) place(x, y int, items []item) int {
	if y < 0 || y >= b.r.H {
		return 0
	}
	s, row := b.s, b.r.Y+y
	col := b.r.X + x
	lo, hi := b.r.X, b.r.X+b.r.W
	n := 0
	for _, it := range items {
		if it.w == 0 {
			continue
		}
		end := col + it.w
		switch {
		case end <= lo:
			// wholly left of the view
		case col < lo:
			// A wide cluster cut by the left edge: its visible half is blank.
			s.put(lo, row, blank(it.id))
			n++
		case col >= hi:
			return n
		case end > hi:
			// A wide cluster at the last column is never split: blank it.
			s.put(col, row, blank(it.id))
			return n + 1
		default:
			s.put(col, row, Cell{Cluster: it.text, Width: uint8(it.w), Style: it.id}) // #nosec G115 -- w is 1 or 2
			if it.w == 2 {
				s.put(col+1, row, Cell{Style: it.id})
			}
			n += it.w
		}
		col = end
	}
	return n
}

// SetString writes the plain text s on row y starting at column x, in style
// id, and returns the number of columns it wrote. It does not wrap: text past
// the right edge, or above or left of b, is clipped. Escape sequences,
// newlines, tabs and other control characters in s are removed. A wide
// cluster never straddles the last column: where it would not fit, its visible
// column is blanked in id instead. Overwriting half of an existing wide
// cluster blanks the other half.
func (b *Buffer) SetString(x, y int, s string, id StyleID) int {
	if isPrintableASCII(s) {
		return b.placeASCII(x, y, s, id)
	}
	return b.place(x, y, b.items(s, id))
}

// placeASCII writes printable ASCII s, one column per byte, without building an
// item slice: the items go through a fixed array, a chunk at a time.
func (b *Buffer) placeASCII(x, y int, s string, id StyleID) int {
	var a [48]item
	n, wrote := 0, 0
	for i := 0; i < len(s); i++ {
		a[n] = item{text: asciiGlyph(s[i]), w: 1, id: id}
		if n++; n == len(a) || i == len(s)-1 {
			wrote += b.place(x+wrote, y, a[:n])
			n = 0
		}
	}
	return wrote
}

// SetRunes is SetString for a rune slice: it writes r on row y from column x in
// style id and returns the columns written. Printable ASCII, the common case, is
// written without building a string.
func (b *Buffer) SetRunes(x, y int, r []rune, id StyleID) int {
	for _, c := range r {
		if c < 0x20 || c >= 0x7f {
			return b.SetString(x, y, string(r), id)
		}
	}
	var a [48]item
	n, wrote := 0, 0
	for i, c := range r {
		a[n] = item{text: asciiGlyph(byte(c)), w: 1, id: id} // #nosec G115 -- c is printable ASCII here, checked by the caller
		if n++; n == len(a) || i == len(r)-1 {
			wrote += b.place(x+wrote, y, a[:n])
			n = 0
		}
	}
	return wrote
}

// asciiBytes holds every byte value once, so a one-column glyph is a slice of
// it and needs no allocation.
const asciiBytes = "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f" +
	"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f" +
	" !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~\x7f"

func asciiGlyph(c byte) string { return asciiBytes[c : c+1] }

// SetStyled writes the styled string s (it may carry SGR sequences and OSC 8
// hyperlinks, but no newline) on row y from column x, adding its styles to the
// style table, and returns the number of columns written. The pen starts as
// the default style. A string the cell grid cannot represent is rejected with
// an error wrapping ErrUnsupported and nothing is written. Clipping and wide
// clusters behave as in SetString.
func (b *Buffer) SetStyled(x, y int, s string) (int, error) {
	items, err := b.styledItems(s)
	if err != nil {
		return 0, err
	}
	return b.place(x, y, items), nil
}

// Fill sets every cell of r (relative to b, clipped to b) to repetitions of
// cluster in style id. An empty or unprintable cluster fills with spaces. A
// wide cluster is placed only whole: a column left over at the right edge of r
// is filled with a blank.
func (b *Buffer) Fill(r Rect, cluster string, id StyleID) {
	v := b.Sub(r)
	if v.r.W == 0 {
		return
	}
	it := item{text: " ", w: 1, id: id}
	if isPrintableASCII(cluster) && len(cluster) == 1 {
		it.text = cluster
	} else if items := b.items(cluster, id); len(items) > 0 {
		it = items[0]
	}
	// One row of items, built in a fixed array when the view is narrow enough.
	var arr [256]item
	row := arr[:0]
	if v.r.W/it.w+1 > len(arr) {
		row = make([]item, 0, v.r.W/it.w+1)
	}
	for x := 0; x < v.r.W; x += it.w {
		row = append(row, it)
	}
	for y := 0; y < v.r.H; y++ {
		v.place(0, y, row)
	}
}

// Lines returns one styled string per row of b. Every row starts from the
// default style and ends with it restored, so rows are independent.
func (b *Buffer) Lines() []string {
	out := make([]string, b.r.H)
	var sb []byte
	for y := range out {
		sb = b.appendRow(sb[:0], y)
		out[y] = string(sb)
	}
	return out
}

// String returns the styled string form of b: its Lines joined by "\n".
// Parse(b.String()) gives back a Buffer with the same cells.
func (b *Buffer) String() string { return strings.Join(b.Lines(), "\n") }

func (b *Buffer) appendRow(dst []byte, y int) []byte {
	var pen Style
	for x := 0; x < b.r.W; x++ {
		c := b.At(x, y)
		if c.Width == 0 {
			continue
		}
		st := b.Style(c.Style)
		dst = appendPen(dst, pen, st)
		pen = st
		dst = append(dst, c.Cluster...)
	}
	return appendPen(dst, pen, Style{})
}

// appendPen appends the bytes that take the terminal from pen from to to.
func appendPen(dst []byte, from, to Style) []byte {
	if from == to {
		return dst
	}
	dst = render.AppendSGR(dst, toRender(from), toRender(to))
	if from.Link != to.Link {
		dst = append(dst, "\x1b]8;;"...)
		dst = append(dst, to.Link...)
		dst = append(dst, 0x1b, '\\')
	}
	return dst
}

func toRender(s Style) render.Style {
	return render.Style{Attrs: uint16(s.Attrs), FG: uint32(s.FG), BG: uint32(s.BG), UL: s.UnderlineStyle}
}

func fromRender(s render.Style) Style {
	return Style{Attrs: Attr(s.Attrs), FG: Color(s.FG), BG: Color(s.BG), UnderlineStyle: s.UL, Link: s.Link}
}

// Parse converts a styled string to a Buffer. Rows are separated by "\n" (a
// trailing "\r" on a row is dropped); the Buffer is as wide as the widest row
// and rows are padded with blanks. Each row starts from the default style, as
// in the cell renderer, so a row that leaves a style open does not affect the
// next. Parse("") is a Buffer of one empty row. A string the cell grid cannot
// represent is rejected with an error wrapping ErrUnsupported that names the
// renderer's reason.
func Parse(s string) (*Buffer, error) { return ParseMeasured(s, ansi.Measurer{}) }

// ParseMeasured is Parse with widths measured by m, which the returned Buffer
// keeps (see SetMeasurer).
func ParseMeasured(s string, m ansi.Measurer) (*Buffer, error) {
	lines := strings.Split(s, "\n")
	rows := make([][]render.Glyph, len(lines))
	w := 0
	for i, ln := range lines {
		ln = strings.TrimSuffix(ln, "\r")
		g, reason, ok := render.ParseGlyphs(ln, m)
		if !ok {
			return nil, fmt.Errorf("%w: row %d: %s", ErrUnsupported, i, reason)
		}
		rows[i] = g
		w = max(w, len(g))
	}
	b := New(w, len(rows))
	b.s.m = m
	for y, g := range rows {
		for x, gl := range g {
			b.s.cells[y*b.s.stride+x] = Cell{Cluster: gl.Text, Width: gl.Width, Style: b.StyleID(fromRender(gl.Style))}
		}
	}
	return b, nil
}

// items splits plain text into placeable clusters in style id.
func (b *Buffer) items(text string, id StyleID) []item {
	if isPrintableASCII(text) {
		out := make([]item, len(text))
		for i := range text {
			out[i] = item{text[i : i+1], 1, id}
		}
		return out
	}
	text = ansi.Sanitize(text) // also drops every control character but "\n"
	text = strings.ReplaceAll(text, "\n", "")
	text = strings.ToValidUTF8(text, "")
	if gs, _, ok := render.ParseGlyphs(text, b.s.m); ok {
		return glyphItems(gs, func(render.Style) StyleID { return id })
	}
	return b.runeItems(text, id)
}

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] >= 0x7f {
			return false
		}
	}
	return true
}

// runeItems is the fallback when the cell parser rejects text whose widths
// disagree: each rune is measured on its own, zero-width runes join the
// cluster before them and runes wider than two columns are dropped.
func (b *Buffer) runeItems(text string, id StyleID) []item {
	var out []item
	for _, r := range text {
		if r >= 0x80 && r < 0xa0 {
			continue
		}
		ch := string(r)
		switch w := b.s.m.Width(ch); {
		case w == 0:
			if n := len(out); n > 0 {
				out[n-1].text += ch
			}
		case w <= 2:
			out = append(out, item{ch, w, id})
		}
	}
	return out
}

func (b *Buffer) styledItems(s string) ([]item, error) {
	if strings.ContainsAny(s, "\r\n") {
		return nil, fmt.Errorf("%w: newline in a single row", ErrUnsupported)
	}
	gs, reason, ok := render.ParseGlyphs(s, b.s.m)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupported, reason)
	}
	return glyphItems(gs, func(st render.Style) StyleID { return b.StyleID(fromRender(st)) }), nil
}

// glyphItems turns parsed cells into items, dropping continuation cells.
func glyphItems(gs []render.Glyph, id func(render.Style) StyleID) []item {
	out := make([]item, 0, len(gs))
	for _, g := range gs {
		if g.Width == 0 {
			continue
		}
		out = append(out, item{g.Text, int(g.Width), id(g.Style)})
	}
	return out
}
