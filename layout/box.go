// Package layout provides simple box-model composition on top of the ansi
// package: padding, borders, and side-by-side or stacked joins. It measures content
// with ansi.Width, so text already wrapped in ansi.Style codes still
// aligns correctly.
package layout

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/basetypes"
)

// Border is the set of characters drawn around a Box.
type Border = basetypes.Border

var (
	normalBorder  = Border{Top: "─", Bottom: "─", Left: "│", Right: "│", TopLeft: "┌", TopRight: "┐", BottomLeft: "└", BottomRight: "┘"}
	roundedBorder = Border{Top: "─", Bottom: "─", Left: "│", Right: "│", TopLeft: "╭", TopRight: "╮", BottomLeft: "╰", BottomRight: "╯"}
	doubleBorder  = Border{Top: "═", Bottom: "═", Left: "║", Right: "║", TopLeft: "╔", TopRight: "╗", BottomLeft: "╚", BottomRight: "╝"}
	thickBorder   = Border{Top: "━", Bottom: "━", Left: "┃", Right: "┃", TopLeft: "┏", TopRight: "┓", BottomLeft: "┗", BottomRight: "┛"}
	asciiBorder   = Border{Top: "-", Bottom: "-", Left: "|", Right: "|", TopLeft: "+", TopRight: "+", BottomLeft: "+", BottomRight: "+"}
)

// NormalBorder draws square corners with thin lines.
func NormalBorder() Border { return normalBorder }

// RoundedBorder draws rounded corners with thin lines.
func RoundedBorder() Border { return roundedBorder }

// DoubleBorder draws double lines.
func DoubleBorder() Border { return doubleBorder }

// ThickBorder draws heavy lines.
func ThickBorder() Border { return thickBorder }

// ASCIIBorder uses only 7-bit ASCII, for terminals or fonts without Unicode
// box-drawing character support.
func ASCIIBorder() Border { return asciiBorder }

func hasBorder(b Border) bool { return b != (Border{}) }

// Box is an immutable, chainable builder for a padded, optionally bordered
// block of text — the structural counterpart to ansi.Style.
type Box struct {
	padTop, padRight, padBottom, padLeft int
	marginT, marginR, marginB, marginL   int
	border                               Border
	borderColor                          ansi.Color    // nil = draw the border in the terminal's default colour
	sides                                [4]ansi.Color // per-side colours set by BorderColors: top, right, bottom, left; nil falls back to borderColor
	gradient                             *[2]ansi.RGB  // set by BorderGradient: the border blends from [0] to [1] and back around the box
	bg                                   ansi.Color    // nil = no background
	width                                int           // 0 = size to content
	height                               int           // 0 = size to content
	clip                                 bool          // width was set by Width: wider lines are cut to it
	rows                                 int           // body rows when fixedRows is set; may be 0
	fixedRows                            bool          // set by exactRows: draw rows body rows whatever the content
	title                                string
	titleAlign                           Align
	noTop, noRight, noBottom, noLeft     bool // border sides switched off by BorderSides
}

// NewBox returns an empty Box: no padding, no border, sized to its content.
func NewBox() Box { return Box{} }

// Padding sets the space between the border (or the box's edge, if there is
// no border) and the content, per side.
func (b Box) Padding(top, right, bottom, left int) Box {
	b.padTop, b.padRight, b.padBottom, b.padLeft = top, right, bottom, left
	return b
}

// PaddingAll sets n as the padding on all four sides.
func (b Box) PaddingAll(n int) Box { return b.Padding(n, n, n, n) }

// Margin sets blank space outside the border, per side. It is not coloured by
// Background. Negative values count as 0.
func (b Box) Margin(top, right, bottom, left int) Box {
	b.marginT, b.marginR, b.marginB, b.marginL = maxInt(top, 0), maxInt(right, 0), maxInt(bottom, 0), maxInt(left, 0)
	return b
}

// Height fixes the content height in rows (padding and border are drawn
// outside it); the zero value sizes the box to its content. Shorter content
// is padded with blank rows and extra rows are cut, so the box is always
// exactly h plus padding and border tall.
func (b Box) Height(h int) Box { b.height = maxInt(h, 0); return b }

// Title sets text drawn in the top border, placed by a: AlignStart near the
// left corner, AlignCenter centred, AlignEnd near the right. It is clipped to
// the border, and ignored when the top side is off or there is no border.
func (b Box) Title(s string, a Align) Box { b.title, b.titleAlign = s, a; return b }

// BorderSides chooses which border sides are drawn; all four are on by
// default. A corner is drawn with its top or bottom edge only when the
// vertical side beside it is on; with that side off the edge simply runs the
// box's width.
func (b Box) BorderSides(top, right, bottom, left bool) Box {
	b.noTop, b.noRight, b.noBottom, b.noLeft = !top, !right, !bottom, !left
	return b
}

// Background fills the box interior (content, padding and the blank fill
// up to Width) with c. The border and margin are left alone. Content that
// resets its own styling keeps the background.
func (b Box) Background(c ansi.Color) Box { b.bg = c; return b }

// Border sets the border style (e.g. layout.NormalBorder()); the zero value
// draws no border.
func (b Box) Border(border Border) Box { b.border = border; return b }

// BorderColor draws the border in c (typically a Theme's BorderColor). Each
// border segment — the top row, each side character, the bottom row — is its
// own styled span, so every line of the result is self-contained. Nil, the
// default, leaves the border uncoloured; a box without a border ignores it.
// Colour changes no dimensions: widths are measured without escape codes.
func (b Box) BorderColor(c ansi.Color) Box { b.borderColor = c; return b }

// BorderColors colours the four sides separately: top, right, bottom, left. A
// nil side falls back to BorderColor. A corner takes the colour of its
// horizontal edge, so the top corners are top-coloured and the bottom corners
// bottom-coloured. BorderGradient takes precedence over both.
func (b Box) BorderColors(top, right, bottom, left ansi.Color) Box {
	b.sides = [4]ansi.Color{top, right, bottom, left}
	return b
}

// Sides of a border, in the order BorderColors takes them.
const (
	sideTop = iota
	sideRight
	sideBottom
	sideLeft
)

// sideColor is the colour of one side: its own, else the box's border colour.
func (b Box) sideColor(side int) ansi.Color {
	if c := b.sides[side]; c != nil {
		return c
	}
	return b.borderColor
}

// BorderGradient draws the border in a colour that changes cell by cell around
// the box: from the colour at the top-left corner it blends to to at the point
// half way round, and back to from where it started, so the loop has no seam.
// Each cell of the border is its own styled span (the title's cells too), so
// every line of the result is self-contained. It takes precedence over
// BorderColor, and a box without a border ignores it. Colour changes no
// dimensions. A terminal below true colour shows the nearest colours.
func (b Box) BorderGradient(from, to ansi.RGB) Box {
	b.gradient = &[2]ansi.RGB{from, to}
	return b
}

// Width sets a fixed content width (padding and border are drawn outside
// it); the zero value sizes the box to its widest content line instead. A
// content line wider than w is clipped to it (ansi.Truncate), so every row of
// the box is exactly w plus padding and border wide.
func (b Box) Width(w int) Box { b.width, b.clip = w, w > 0; return b }

// minWidth is Width for the layout helpers that pad blocks to a column: the
// width is a minimum, and content wider than it is kept whole rather than
// clipped (the package's no-truncation convention for FlexRow and Grid).
func (b Box) minWidth(w int) Box { b.width, b.clip = w, false; return b }

// exactRows fixes the number of body rows at n, which may be 0: a frame with
// nothing inside. Height cannot say that, because its 0 means "as many rows
// as the content has", and empty content still counts as one row.
func (b Box) exactRows(n int) Box { b.rows, b.fixedRows = n, true; return b }

// edge returns the horizontal line of a border edge of n cells, with title
// embedded when it is the top edge.
func (b Box) edge(line string, n int, title string) string {
	if title == "" || n <= 0 {
		return strings.Repeat(line, n)
	}
	t := ansi.Truncate(" "+title+" ", n)
	rem := n - ansi.Width(t)
	lead := 0
	switch b.titleAlign {
	case AlignCenter:
		lead = rem / 2
	case AlignEnd:
		lead = rem - minInt(1, rem)
	default:
		lead = minInt(1, rem)
	}
	return strings.Repeat(line, lead) + t + strings.Repeat(line, rem-lead)
}

func minInt(a, b int) int {
	if b < a {
		return b
	}
	return a
}

// corner returns the corner glyph c, or nothing when the vertical side it
// joins is off (there is no column for it to sit in).
func corner(c string, vertOn bool) string {
	if vertOn {
		return c
	}
	return ""
}

// render lays content out inside the box: content lines are padded flush
// to a common width, then padding rows/columns and an optional border are
// added around them. BoxNode draws through it; the exported way to draw a box
// is BoxNode(box, Block(content)) or BoxNode with any Node child.
func (b Box) render(content string) string {
	r := b.plan(content)
	rowCount := b.marginT + b.padTop + r.bodyRows + b.padBottom + b.marginB + 2
	r.out.Grow(rowCount*(r.innerWidth+len(r.left)+len(r.right)+b.marginL+b.marginR+1) + len(content))
	r.margins(b.marginT)
	r.top()
	r.body(content)
	r.bottom()
	r.margins(b.marginB)
	return r.out.String()
}

// boxRender is one render of a Box: the sizes worked out from the content, the
// painters for the border and background, and the output so far. The steps
// top, body and bottom each write their rows to out.
type boxRender struct {
	b Box
	// contentWidth is the width of the text column, innerWidth that plus the
	// side padding; bodyRows is the number of text rows.
	contentWidth, innerWidth, bodyRows int
	// show says which sides of the border are drawn.
	showTop, showBottom, showLeft, showRight bool
	// left and right are the painted side glyphs, used when there is no ring.
	left, right string
	// paint colours border text, one painter per side (see sideTop and the
	// rest); fillBG and reBG apply the background.
	paint  [4]func(string) string
	fillBG func(int) string
	reBG   func(string) string
	// ring paints a gradient border a cell at a time; off when it is not on.
	ring borderRing
	// fullWidth is the width of a margin-only row.
	fullWidth int

	out     strings.Builder
	first   bool // no row written yet: the next one needs no newline before it
	sideRow int  // body rows written so far, the ring's place on the sides
}

// plan measures content against the box and sets up the painters.
func (b Box) plan(content string) *boxRender {
	r := &boxRender{b: b, first: true}
	r.contentWidth = b.width
	nLines := 0
	for rest, more := content, true; more; nLines++ {
		var l string
		l, rest, more = nextLine(rest)
		if b.width == 0 {
			if w := ansi.Width(l); w > r.contentWidth {
				r.contentWidth = w
			}
		}
	}
	bordered := hasBorder(b.border)
	r.showTop, r.showBottom = bordered && !b.noTop, bordered && !b.noBottom
	r.showLeft, r.showRight = bordered && !b.noLeft, bordered && !b.noRight
	if b.width == 0 && r.showTop && b.title != "" {
		// An auto-sized box grows to fit its title (with a space each side).
		need := ansi.Width(b.title) + 2 - b.padLeft - b.padRight
		if need > r.contentWidth {
			r.contentWidth = need
		}
	}
	r.bodyRows = nLines
	if b.height > 0 {
		r.bodyRows = b.height
	}
	if b.fixedRows {
		r.bodyRows = b.rows
	}
	r.innerWidth = r.contentWidth + b.padLeft + b.padRight

	for side := range r.paint {
		r.paint[side] = func(s string) string { return s }
		if c := b.sideColor(side); bordered && c != nil {
			style := ansi.NewStyle().Foreground(c)
			r.paint[side] = func(s string) string {
				if s == "" {
					return s
				}
				return style.Render(s)
			}
		}
	}
	// fillBG paints interior spaces; reBG re-opens the background after a
	// reset inside content.
	r.fillBG = func(n int) string { return strings.Repeat(" ", maxInt(n, 0)) }
	r.reBG = func(s string) string { return s }
	if b.bg != nil {
		style := ansi.NewStyle().Background(b.bg)
		r.fillBG = func(n int) string {
			if n <= 0 {
				return ""
			}
			return style.Render(strings.Repeat(" ", n))
		}
		open, _, _ := strings.Cut(style.Render("X"), "X")
		r.reBG = func(s string) string {
			if s == "" {
				return s
			}
			s = strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+open)
			s = strings.ReplaceAll(s, "\x1b[m", "\x1b[m"+open)
			return open + s + "\x1b[0m"
		}
	}

	if r.showLeft {
		r.left = r.paint[sideLeft](b.border.Left)
	}
	if r.showRight {
		r.right = r.paint[sideRight](b.border.Right)
	}
	// With a gradient the border is painted a cell at a time along the loop
	// top (left to right), right side (down), bottom (right to left) and left
	// side (up). ring says how long the loop is, so a cell's place is a fraction.
	if bordered && b.gradient != nil {
		topW := ansi.Width(corner(b.border.TopLeft, r.showLeft)) + r.innerWidth + ansi.Width(corner(b.border.TopRight, r.showRight))
		botW := ansi.Width(corner(b.border.BottomLeft, r.showLeft)) + r.innerWidth + ansi.Width(corner(b.border.BottomRight, r.showRight))
		r.ring = borderRing{
			from: b.gradient[0], to: b.gradient[1],
			topW: topW, botW: botW,
			side: b.padTop + r.bodyRows + b.padBottom,
		}
		if !r.showTop {
			r.ring.topW = 0
		}
		if !r.showBottom {
			r.ring.botW = 0
		}
		for side := range r.paint {
			r.paint[side] = func(s string) string { return s } // the ring paints the cells itself
		}
	}
	r.fullWidth = b.marginL + r.innerWidth + b.marginR
	if r.showLeft {
		r.fullWidth += ansi.Width(b.border.Left)
	}
	if r.showRight {
		r.fullWidth += ansi.Width(b.border.Right)
	}
	return r
}

// newRow starts the next output row.
func (r *boxRender) newRow() {
	if !r.first {
		r.out.WriteByte('\n')
	}
	r.first = false
}

// margins writes n blank margin rows.
func (r *boxRender) margins(n int) {
	for i := 0; i < n; i++ {
		r.newRow()
		writeSpaces(&r.out, r.fullWidth)
	}
}

// top writes the top border row, when there is one.
func (r *boxRender) top() {
	if !r.showTop {
		return
	}
	b := r.b
	r.newRow()
	writeSpaces(&r.out, b.marginL)
	topRow := corner(b.border.TopLeft, r.showLeft) + b.edge(b.border.Top, r.innerWidth, b.title) + corner(b.border.TopRight, r.showRight)
	if r.ring.on() {
		r.out.WriteString(r.ring.paintTop(topRow))
	} else {
		r.out.WriteString(r.paint[sideTop](topRow))
	}
	writeSpaces(&r.out, b.marginR)
}

// bottom writes the bottom border row, when there is one.
func (r *boxRender) bottom() {
	if !r.showBottom {
		return
	}
	b := r.b
	r.newRow()
	writeSpaces(&r.out, b.marginL)
	botRow := corner(b.border.BottomLeft, r.showLeft) + strings.Repeat(b.border.Bottom, r.innerWidth) + corner(b.border.BottomRight, r.showRight)
	if r.ring.on() {
		r.out.WriteString(r.ring.paintBottom(botRow))
	} else {
		r.out.WriteString(r.paint[sideBottom](botRow))
	}
	writeSpaces(&r.out, b.marginR)
}

// body writes the padding rows and the content rows between them.
func (r *boxRender) body(content string) {
	for i := 0; i < r.b.padTop; i++ {
		r.row("")
	}
	written := 0
	for rest, more := content, true; more && written < r.bodyRows; written++ {
		var l string
		l, rest, more = nextLine(rest)
		r.row(l)
	}
	for ; written < r.bodyRows; written++ {
		r.row("")
	}
	for i := 0; i < r.b.padBottom; i++ {
		r.row("")
	}
}

// row writes one body row: margin, side border, left pad, text, fill to
// contentWidth, right pad, side border, margin.
func (r *boxRender) row(text string) {
	b := r.b
	r.newRow()
	w := ansi.Width(text)
	if b.clip && w > r.contentWidth {
		// A fixed Width clips: a wider line would push the right border
		// out of line and make the box wider than asked.
		text = ansi.Truncate(text, r.contentWidth)
		w = ansi.Width(text)
	}
	writeSpaces(&r.out, b.marginL)
	if r.ring.on() {
		if r.showLeft {
			r.out.WriteString(r.ring.paintSide(b.border.Left, r.sideRow, false))
		}
	} else {
		r.out.WriteString(r.left)
	}
	r.out.WriteString(r.fillBG(b.padLeft))
	r.out.WriteString(r.reBG(text))
	r.out.WriteString(r.fillBG(r.contentWidth - w + b.padRight))
	if r.ring.on() {
		if r.showRight {
			r.out.WriteString(r.ring.paintSide(b.border.Right, r.sideRow, true))
		}
	} else {
		r.out.WriteString(r.right)
	}
	r.sideRow++
	writeSpaces(&r.out, b.marginR)
}

// Align is a cross-axis alignment for joinHorizontalAlign and
// joinVerticalAlign. The zero value, AlignStart, reproduces the behaviour of
// joinHorizontal (top-aligned) and joinVertical (left-aligned).
type Align int

const (
	// AlignStart top-aligns blocks in joinHorizontalAlign (blank rows are
	// added below each block's content) and left-aligns lines in
	// joinVerticalAlign (lines are right-padded to the common width). It is
	// the zero value, and matches joinHorizontal/joinVertical.
	AlignStart Align = iota
	// AlignCenter vertically centers each block's rows (joinHorizontalAlign)
	// or horizontally centers each line (joinVerticalAlign) within the
	// available space. When the padding is odd, the extra row or column goes
	// after/on the right, matching AlignStart's placement of extra space.
	AlignCenter
	// AlignEnd bottom-aligns blocks in joinHorizontalAlign (blank rows are
	// added above each block's content) and right-aligns lines in
	// joinVerticalAlign (lines are left-padded to the common width).
	AlignEnd
)

// joinHorizontal lays out blocks of (possibly multi-line) text side by
// side, top-aligned, separated by gap spaces. Blocks with fewer lines than
// the tallest one are padded with blank rows so every block contributes
// the same number of rows to the result. It is joinHorizontalAlign with
// AlignStart.
func joinHorizontal(gap int, blocks ...string) string {
	return joinHorizontalAlign(gap, AlignStart, blocks...)
}

// joinHorizontalAlign is joinHorizontal with an explicit cross-axis
// (vertical) alignment: AlignStart top-aligns each block (as joinHorizontal
// does), AlignCenter vertically centers it, and AlignEnd bottom-aligns it,
// within the row height of the tallest block.
func joinHorizontalAlign(gap int, align Align, blocks ...string) string {
	if len(blocks) == 0 {
		return ""
	}

	n := len(blocks)
	ints := make([]int, 3*n) // one allocation for the three per-block tables
	counts, widths := ints[:n:n], ints[n:2*n:2*n]
	minWs := ints[2*n:] // narrowest line; == widths[i] means every line is full width
	rests := make([]string, len(blocks))
	maxLines := 0
	total := 0
	for i, blk := range blocks {
		rests[i] = blk
		total += len(blk)
		minWs[i] = -1
		for rest, more := blk, true; more; counts[i]++ {
			var l string
			l, rest, more = nextLine(rest)
			w := ansi.Width(l)
			if w > widths[i] {
				widths[i] = w
			}
			if minWs[i] < 0 || w < minWs[i] {
				minWs[i] = w
			}
		}
		if counts[i] > maxLines {
			maxLines = counts[i]
		}
	}

	// offsets[i] is the number of blank rows placed above block i's content.
	offsets := make([]int, len(blocks))
	for i, n := range counts {
		missing := maxLines - n
		switch align {
		case AlignCenter:
			offsets[i] = missing / 2
		case AlignEnd:
			offsets[i] = missing
		default: // AlignStart
			offsets[i] = 0
		}
	}

	if gap < 0 {
		gap = 0 // like joinVertical, a gap <= 0 means none
	}
	sumW := gap * (len(blocks) - 1)
	for _, w := range widths {
		sumW += w
	}
	var out strings.Builder
	out.Grow(maxLines*(sumW+1) + total)
	for row := 0; row < maxLines; row++ {
		if row > 0 {
			out.WriteByte('\n')
		}
		for i := range blocks {
			cell, real := "", false
			if r := row - offsets[i]; r >= 0 && r < counts[i] {
				cell, rests[i], _ = nextLine(rests[i])
				real = true
			}
			out.WriteString(cell)
			switch {
			case !real:
				writeSpaces(&out, widths[i])
			case minWs[i] == widths[i]:
				// every line already fills the block: no re-measure
			default:
				writeSpaces(&out, widths[i]-ansi.Width(cell))
			}
			if i < len(blocks)-1 {
				writeSpaces(&out, gap)
			}
		}
	}
	return out.String()
}

// joinVertical stacks blocks of (possibly multi-line) text on top of each
// other, left-aligned, with gap blank lines between adjacent blocks
// (gap <= 0 for none) — the vertical counterpart to joinHorizontal. Every
// line, blank separators included, is right-padded with spaces to the width
// of the widest line across all blocks, so the result is a clean rectangle
// that can itself be boxed or joined horizontally. An empty block counts as
// one blank line. Widths are measured with ansi.Width, so styled and
// wide-rune content aligns; line content is never altered.
func joinVertical(gap int, blocks ...string) string {
	return joinVerticalAlign(gap, AlignStart, blocks...)
}

// joinVerticalAlign is joinVertical with an explicit cross-axis
// (horizontal) alignment: AlignStart left-aligns each line (as joinVertical
// does, right-padding with spaces), AlignCenter horizontally centers it, and
// AlignEnd right-aligns it (left-padding with spaces), all to the common
// width of the widest line across every block.
func joinVerticalAlign(gap int, align Align, blocks ...string) string {
	if len(blocks) == 0 {
		return ""
	}

	width, nLines, total := 0, 0, 0
	minW := -1 // narrowest line; == width means every real line fills it
	for i, blk := range blocks {
		if i > 0 && gap > 0 {
			nLines += gap
		}
		total += len(blk)
		for rest, more := blk, true; more; nLines++ {
			var l string
			l, rest, more = nextLine(rest)
			w := ansi.Width(l)
			if w > width {
				width = w
			}
			if minW < 0 || w < minW {
				minW = w
			}
		}
	}
	uniform := minW == width

	var out strings.Builder
	out.Grow(nLines*width + nLines + total)
	first := true
	line := func(l string) {
		if !first {
			out.WriteByte('\n')
		}
		first = false
		pad := 0
		if l == "" {
			pad = width // a blank line (or empty block) is padded to the full width
		} else if !uniform {
			pad = width - ansi.Width(l)
			if pad < 0 {
				pad = 0
			}
		}
		switch align {
		case AlignCenter:
			writeSpaces(&out, pad/2)
			out.WriteString(l)
			writeSpaces(&out, pad-pad/2)
		case AlignEnd:
			writeSpaces(&out, pad)
			out.WriteString(l)
		default: // AlignStart
			out.WriteString(l)
			writeSpaces(&out, pad)
		}
	}
	for i, blk := range blocks {
		if i > 0 {
			for g := 0; g < gap; g++ {
				line("")
			}
		}
		for rest, more := blk, true; more; {
			var l string
			l, rest, more = nextLine(rest)
			line(l)
		}
	}
	return out.String()
}

// spaces is a run of blanks long enough that most padding is one WriteString.
const spaces = "                                                                "

// writeSpaces writes n spaces (nothing if n <= 0) without allocating a
// string of them.
func writeSpaces(b *strings.Builder, n int) {
	for n > len(spaces) {
		b.WriteString(spaces)
		n -= len(spaces)
	}
	if n > 0 {
		b.WriteString(spaces[:n])
	}
}
