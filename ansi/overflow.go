package ansi

import "strings"

// TruncateStart cuts s to at most width columns by dropping its start and
// putting glyph there, so what remains is the end of the string: the file name
// of a path. glyph may be empty. Styling that was open at the cut stays in
// force over the end, as TrimLeftWidth keeps it. A glyph wider than the width
// is cut to fit.
func TruncateStart(s string, width int, glyph string) string {
	if width <= 0 {
		return ""
	}
	w := Width(s)
	if w <= width {
		return s
	}
	glyph = Truncate(glyph, width)
	return glyph + TrimLeftWidth(s, w-(width-Width(glyph)))
}

// TruncateMiddle cuts s to at most width columns by dropping the middle and
// putting glyph there, so both the beginning and the end survive: a long path
// or an identifier. glyph may be empty. When the room is odd the extra column
// goes to the start.
func TruncateMiddle(s string, width int, glyph string) string {
	if width <= 0 {
		return ""
	}
	w := Width(s)
	if w <= width {
		return s
	}
	glyph = Truncate(glyph, width)
	room := width - Width(glyph)
	head := room - room/2
	tail := room / 2
	return Truncate(s, head) + glyph + TrimLeftWidth(s, w-tail)
}

// Justify widens the gaps between the words of line, left to right, until it
// is width columns wide, the way justified text is set. A space inside an
// escape sequence is not a gap. A line with no gap, one already as wide as
// width or wider, or one with leading or trailing spaces, is returned as it
// is. The extra spaces are spread evenly, the first gaps taking the remainder.
func Justify(line string, width int) string {
	w := Width(line)
	if w >= width || line == "" || line[0] == ' ' || line[len(line)-1] == ' ' {
		return line
	}
	var gaps []int // byte offsets of the spaces that are gaps
	for i := 0; i < len(line); {
		if line[i] == 0x1b {
			if j, _ := escEnd(line, i); j > i {
				i = j
				continue
			}
		}
		if line[i] == ' ' {
			gaps = append(gaps, i)
		}
		i++
	}
	if len(gaps) == 0 {
		return line
	}
	extra := width - w
	each, rem := extra/len(gaps), extra%len(gaps)
	var b strings.Builder
	last := 0
	for k, g := range gaps {
		b.WriteString(line[last:g])
		b.WriteString(strings.Repeat(" ", 1+each))
		if k < rem {
			b.WriteByte(' ')
		}
		last = g + 1
	}
	b.WriteString(line[last:])
	return b.String()
}
