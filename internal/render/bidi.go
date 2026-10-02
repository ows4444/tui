package render

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/bidi"
)

// BidiLine returns line with its characters in visual order per the Unicode
// Bidirectional Algorithm (UAX #9), taking the paragraph direction from the first
// strong character, for terminals that draw text in logical order. Each
// character keeps its own style and hyperlink, and a character at an odd level
// that has an exact mirror image (a bracket) is replaced by it.
//
// A line with no right-to-left or formatting character, and a line the cell
// parser declines (graphics, malformed escapes), is returned unchanged, byte for
// byte. The reordering is for display only; the caller keeps the logical text.
func BidiLine(line string, m ansi.Measurer) string {
	if !mayNeedBidi(line) {
		return line
	}
	glyphs, _, ok := ParseGlyphs(line, m)
	if !ok {
		return line
	}
	type unit struct{ first, end int } // glyphs[first:end]: a head cell and its continuations
	var units []unit
	var reps []rune
	for i := 0; i < len(glyphs); {
		j := i + 1
		for j < len(glyphs) && glyphs[j].Width == 0 {
			j++
		}
		r := ' '
		for _, c := range glyphs[i].Text {
			r = c
			break
		}
		units = append(units, unit{i, j})
		reps = append(reps, r)
		i = j
	}
	res := bidi.Resolve(reps, bidi.Auto)
	moved := false
	for k, o := range res.Order {
		if k != o {
			moved = true
			break
		}
	}
	if !moved {
		mirrored := false
		for k, r := range reps {
			if res.Levels[k]&1 == 1 {
				if _, ok := bidi.Mirror(r); ok {
					mirrored = true
					break
				}
			}
		}
		if !mirrored {
			return line
		}
	}

	var out []byte
	var pen Style
	flushPen := func(to Style) {
		out = AppendSGR(out, pen, to)
		if to.Link != pen.Link {
			if to.Link == "" {
				out = append(out, "\x1b]8;;\x1b\\"...)
			} else {
				out = append(out, "\x1b]8;;"...)
				out = append(out, to.Link...)
				out = append(out, "\x1b\\"...)
			}
		}
		pen = to
	}
	for _, o := range res.Order {
		u := units[o]
		head := glyphs[u.first]
		if head.Style != pen {
			flushPen(head.Style)
		}
		text := head.Text
		if res.Levels[o]&1 == 1 {
			if r := []rune(text); len(r) == 1 {
				if mr, ok := bidi.Mirror(r[0]); ok {
					text = string(mr)
				}
			}
		}
		out = append(out, text...)
	}
	if pen != (Style{}) {
		flushPen(Style{})
	}
	return string(out)
}

// mayNeedBidi is a cheap scan for a rune that can make the algorithm reorder:
// a right-to-left letter or Arabic number, or an embedding, override or isolate
// control. Without one every level is 0 and the line stays as it is.
func mayNeedBidi(line string) bool {
	for _, r := range line {
		if r < 0x0590 {
			continue
		}
		switch bidi.ClassOf(r) {
		case bidi.R, bidi.AL, bidi.AN, bidi.LRE, bidi.LRO, bidi.RLE, bidi.RLO, bidi.LRI, bidi.RLI, bidi.FSI:
			return true
		}
	}
	return false
}
