package ansi

import "strings"

// Span renders the concatenation of parts in this style, re-emitting the
// style after every reset inside them. Unlike Render, nesting works: in
// outer.Span("a ", inner.Render("b"), " c") the text " c" is still in the
// outer style even though inner.Render ended with a full reset. Multi-line
// text is styled per line, as Render does. A style with no attributes
// returns the parts joined unchanged.
func (s Style) Span(parts ...string) string {
	text := strings.Join(parts, "")
	if s.link != "" {
		return s.linkEach(text, func(st Style, line string) string { return st.Span(line) })
	}
	var buf [96]byte
	seq := string(s.appendSequence(buf[:0]))
	if seq == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		line = strings.ReplaceAll(line, Reset, Reset+seq)
		line = strings.ReplaceAll(line, "\x1b[m", "\x1b[m"+seq)
		// A reset at the very end needs no re-opening.
		if trimmed := strings.TrimSuffix(line, seq); trimmed != line {
			lines[i] = seq + trimmed
			continue
		}
		lines[i] = seq + line + Reset
	}
	return strings.Join(lines, "\n")
}
