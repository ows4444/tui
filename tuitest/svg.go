package tuitest

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/vtscreen"
)

// SVGOptions shapes Session.SVG. Every field is optional.
type SVGOptions struct {
	// Title, when set, draws a title bar with it above the screen, like a
	// terminal window.
	Title string
	// Foreground and Background are the colours of a cell that sets none. The
	// zero value of each is a light grey (#cccccc) on near black (#1e1e1e).
	Foreground, Background *ansi.RGB
	// Palette replaces the 16 ANSI colours; nil keeps the built-in one.
	Palette *[16]ansi.RGB
	// FontFamily is the monospace font (default a monospace stack); FontSize is
	// its size in pixels (default 14).
	FontFamily string
	FontSize   int
	// Cursor draws the cursor cell when the program shows it.
	Cursor bool
	// TrimRows drops the blank rows at the bottom, so the image is as tall as
	// what was drawn (at least one row).
	TrimRows bool
}

// defaultPalette is the 16 ANSI colours of a dark editor terminal, as 0xRRGGBB.
var defaultPalette = func() [16]ansi.RGB {
	hex := [16]uint32{
		0x000000, 0xcd3131, 0x0dbc79, 0xe5e510, 0x2472c8, 0xbc3fbc, 0x11a8cd, 0xe5e5e5,
		0x666666, 0xf14c4c, 0x23d18b, 0xf5f543, 0x3b8eea, 0xd670d6, 0x29b8db, 0xe5e5e5,
	}
	var p [16]ansi.RGB
	for i, h := range hex {
		p[i] = ansi.RGB{R: uint8(h >> 16), G: uint8(h >> 8), B: uint8(h)} // #nosec G115 -- masked by the shifts of a 24-bit value
	}
	return p
}()

// SVG draws the screen as a standalone SVG image: each cell's text, colours
// and bold, dim, italic, underline, reverse and strike attributes, on the
// screen's size, so a widget can be shown in documentation or a pull request
// without a screenshot tool. One limit: the emulator keeps one rune per cell
// (see Cell), so a wide character takes one column here, not two. Every glyph is placed at its own column, so the
// layout holds in any font. The result is XML and has no external references.
func (s *Session) SVG(o SVGOptions) string {
	s.mu.Lock()
	w, h := s.w, s.h
	s.mu.Unlock()
	cells := make([][]Cell, h)
	for y := range cells {
		cells[y] = make([]Cell, w)
		for x := range cells[y] {
			cells[y][x] = s.Cell(x, y)
		}
	}
	if o.TrimRows {
		keep := 1
		for y, row := range cells {
			for _, c := range row {
				if (c.Grapheme != " " && c.Grapheme != "") || c.BG != nil || c.Reverse {
					keep = y + 1
					break
				}
			}
		}
		cells = cells[:keep]
	}
	cx, cy, cursor := s.Cursor()
	return renderSVG(cells, cx, cy, cursor && o.Cursor, o)
}

func renderSVG(cells [][]Cell, cx, cy int, showCursor bool, o SVGOptions) string {
	size := o.FontSize
	if size <= 0 {
		size = 14
	}
	family := o.FontFamily
	if family == "" {
		family = "ui-monospace, SFMono-Regular, Menlo, Consolas, 'Liberation Mono', monospace"
	}
	pal := &defaultPalette
	if o.Palette != nil {
		pal = o.Palette
	}
	fg, bg := ansi.RGB{R: 0xcc, G: 0xcc, B: 0xcc}, ansi.RGB{R: 0x1e, G: 0x1e, B: 0x1e}
	if o.Foreground != nil {
		fg = *o.Foreground
	}
	if o.Background != nil {
		bg = *o.Background
	}

	cw := float64(size) * 0.6
	lh := float64(size) * 1.4
	pad := float64(size)
	rows := len(cells)
	cols := 0
	if rows > 0 {
		cols = len(cells[0])
	}
	bar := 0.0
	if o.Title != "" {
		bar = lh + 8
	}
	width := float64(cols)*cw + 2*pad
	height := float64(rows)*lh + 2*pad + bar

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f" role="img">`+"\n", width, height, width, height)
	fmt.Fprintf(&b, `<rect width="%.0f" height="%.0f" rx="6" fill="%s"/>`+"\n", width, height, hexOf(bg))
	if o.Title != "" {
		fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="5" fill="#ff5f56"/><circle cx="%.1f" cy="%.1f" r="5" fill="#ffbd2e"/><circle cx="%.1f" cy="%.1f" r="5" fill="#27c93f"/>`+"\n",
			pad, bar/2, pad+18, bar/2, pad+36, bar/2)
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" font-family="%s" font-size="%d" fill="%s" fill-opacity="0.7" text-anchor="middle">%s</text>`+"\n",
			width/2, bar/2+float64(size)*0.35, escapeXML(family), size, hexOf(fg), escapeXML(o.Title))
	}
	fmt.Fprintf(&b, `<g font-family="%s" font-size="%d" xml:space="preserve">`+"\n", escapeXML(family), size)

	// Backgrounds first, one rect per run of equal colour, then the text.
	type look struct {
		fg, bg                                        ansi.RGB
		bold, dim, italic, underline, strike, hasText bool
	}
	lookOf := func(c Cell) look {
		f, g := colorOrDefault(c.FG, fg, pal), colorOrDefault(c.BG, bg, pal)
		if c.Reverse {
			f, g = g, f
		}
		return look{fg: f, bg: g, bold: c.Bold, dim: c.Dim, italic: c.Italic, underline: c.Underline, strike: c.Strike}
	}
	for y, row := range cells {
		top := pad + bar + float64(y)*lh
		for x := 0; x < len(row); {
			l := lookOf(row[x])
			end := x + 1
			for end < len(row) && lookOf(row[end]).bg == l.bg {
				end++
			}
			if l.bg != bg {
				fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`+"\n",
					pad+float64(x)*cw, top, float64(end-x)*cw, lh, hexOf(l.bg))
			}
			x = end
		}
	}
	if showCursor && cy >= 0 && cy < rows && cx >= 0 && cx < cols {
		fmt.Fprintf(&b, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s" fill-opacity="0.55"/>`+"\n",
			pad+float64(cx)*cw, pad+bar+float64(cy)*lh, cw, lh, hexOf(fg))
	}
	for y, row := range cells {
		base := pad + bar + float64(y)*lh + lh*0.72
		for x := 0; x < len(row); {
			l := lookOf(row[x])
			end := x + 1
			for end < len(row) {
				n := lookOf(row[end])
				n.bg = l.bg
				if n != l {
					break
				}
				end++
			}
			// Each glyph gets its own x, so columns hold in any font; blanks are
			// skipped unless an underline or strike has to run through them.
			keepBlanks := l.underline || l.strike
			var text strings.Builder
			var xs []string
			for k := x; k < end; k++ {
				t := cellText(row[k])
				if t == "" || (!keepBlanks && strings.TrimSpace(t) == "") {
					continue
				}
				text.WriteString(escapeXML(t))
				xs = append(xs, fmt.Sprintf("%.2f", pad+float64(k)*cw))
			}
			if len(xs) > 0 {
				attrs := ""
				if l.bold {
					attrs += ` font-weight="bold"`
				}
				if l.italic {
					attrs += ` font-style="italic"`
				}
				switch {
				case l.underline && l.strike:
					attrs += ` text-decoration="underline line-through"`
				case l.underline:
					attrs += ` text-decoration="underline"`
				case l.strike:
					attrs += ` text-decoration="line-through"`
				}
				if l.dim {
					attrs += ` fill-opacity="0.6"`
				}
				fmt.Fprintf(&b, `<text x="%s" y="%.2f" fill="%s"%s>%s</text>`+"\n",
					strings.Join(xs, " "), base, hexOf(l.fg), attrs, text.String())
			}
			x = end
		}
	}
	b.WriteString("</g>\n</svg>\n")
	return b.String()
}

// cellText is the text one cell contributes: its grapheme, a space for a blank,
// and nothing for the empty cell after a wide character.
func cellText(c Cell) string {
	switch c.Grapheme {
	case "", "\x00":
		return ""
	}
	return c.Grapheme
}

func hexOf(c ansi.RGB) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

func escapeXML(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		case r == '"':
			b.WriteString("&quot;")
		case r == '\'':
			b.WriteString("&apos;")
		case r < 0x20 && r != '\t' && r != '\n', r == 0x7f, r == 0xfffe, r == 0xffff:
			b.WriteByte(' ') // not allowed in XML 1.0
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// colorOrDefault resolves c to RGB, or def for the default colour.
func colorOrDefault(c ansi.Color, def ansi.RGB, pal *[16]ansi.RGB) ansi.RGB {
	switch v := c.(type) {
	case ansi.RGB:
		return v
	case ansi.BasicColor:
		if int(v) < len(pal) {
			return pal[v]
		}
	case ansi.Color256:
		n := int(v)
		switch {
		case n < 16:
			return pal[n]
		case n < 232:
			n -= 16
			lv := func(i int) uint8 {
				if i == 0 {
					return 0
				}
				return uint8(55 + 40*i) // #nosec G115 -- i is 1..5, so at most 255
			}
			return ansi.RGB{R: lv(n / 36), G: lv(n / 6 % 6), B: lv(n % 6)}
		default:
			l := uint8(8 + 10*(n-232)) // #nosec G115 -- n is 232..255, so at most 238
			return ansi.RGB{R: l, G: l, B: l}
		}
	}
	return def
}

// SVGOf draws styled text, such as tui.Screenshot.ANSI, as an SVG image the way
// Session.SVG draws a session's screen: the rows are laid onto a w by h
// terminal, colours and attributes kept. It is how a running program's
// screenshot becomes a picture for the docs.
func SVGOf(styled string, w, h int, o SVGOptions) string {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	scr := vtscreen.NewScreen(w, h)
	scr.Write([]byte(strings.ReplaceAll(styled, "\n", "\r\n")))
	cells := make([][]Cell, h)
	for y := range cells {
		cells[y] = make([]Cell, w)
		for x := range cells[y] {
			c := scr.Cell(x, y)
			cells[y][x] = Cell{
				Grapheme: string(c.Rune), Bold: c.Bold, Dim: c.Dim, Italic: c.Italic,
				Underline: c.Underline, Reverse: c.Reverse, Strike: c.Strike, FG: c.FG, BG: c.BG,
			}
		}
	}
	if o.TrimRows {
		keep := 1
		for y, row := range cells {
			for _, c := range row {
				if (c.Grapheme != " " && c.Grapheme != "") || c.BG != nil || c.Reverse {
					keep = y + 1
					break
				}
			}
		}
		cells = cells[:keep]
	}
	return renderSVG(cells, 0, 0, false, o)
}
