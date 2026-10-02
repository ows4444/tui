package render

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// The glyph API lets the public cellbuf package reuse the cell parser and the
// SGR writer without sharing the renderer's interned ids or its Cells state.

// Style is the decoded pen of a cell with the colours and attributes in the
// renderer's own encoding (see cellStyle). Link is the hyperlink target URI,
// "" for none. It is comparable.
type Style struct {
	Attrs  uint16
	FG, BG uint32
	UL     uint8
	Link   string
}

// Glyph is one parsed cell: a head cell carries its cluster text and width
// (1 or 2); the continuation cell of a wide cluster has Width 0 and no Text.
type Glyph struct {
	Text  string
	Width uint8
	Style Style
}

// ParseGlyphs parses one view line (no newline) into cells with the same rules
// the cell renderer applies, measuring widths with m. ok is false, with the
// renderer's fallback slug in reason, when the line uses something the grid
// cannot represent. Trailing blank cells are kept.
func ParseGlyphs(line string, m ansi.Measurer) (glyphs []Glyph, reason string, ok bool) {
	c := New()
	c.measure = m
	c.keepBlank = true
	row, ok := c.parseRow(line, nil)
	if !ok {
		return nil, c.reason, false
	}
	glyphs = make([]Glyph, len(row))
	for i, cl := range row {
		g := Glyph{Width: cl.w, Style: c.exportStyle(c.styles[cl.st])}
		if cl.w != 0 {
			g.Text = c.text(cl)
		}
		glyphs[i] = g
	}
	return glyphs, "", true
}

func (c *Cells) exportStyle(s cellStyle) Style {
	return Style{
		Attrs: s.attrs, FG: uint32(s.fg), BG: uint32(s.bg), UL: s.ul,
		Link: linkURI(c.links[s.link]),
	}
}

// linkURI extracts the target from a raw OSC 8 open sequence
// (ESC ] 8 ; params ; uri ST|BEL); "" for index 0.
func linkURI(raw string) string {
	if raw == "" {
		return ""
	}
	raw = strings.TrimPrefix(raw, "\x1b]8;")
	raw = strings.TrimSuffix(strings.TrimSuffix(raw, "\x1b\\"), "\x07")
	if i := strings.IndexByte(raw, ';'); i >= 0 {
		return raw[i+1:]
	}
	return ""
}

// AppendSGR appends the SGR bytes that take the terminal from pen from to pen
// to (nothing when the SGR state is equal). Links are not part of SGR and are
// ignored.
func AppendSGR(dst []byte, from, to Style) []byte {
	e := cellEmitter{out: dst}
	e.setSGR(importStyle(from), importStyle(to))
	return e.out
}

func importStyle(s Style) cellStyle {
	return cellStyle{attrs: s.Attrs, fg: cellColor(s.FG), bg: cellColor(s.BG), ul: s.UL}
}
