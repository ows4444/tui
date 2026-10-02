package ansi

import "strings"

// Wrap word-wraps s to width visible columns, preserving existing
// newlines as paragraph breaks — each paragraph is wrapped independently,
// rather than treating the whole string as one run of words that could
// merge separate paragraphs together. A single word longer than width is
// never split (there's no hyphenation), so a line can still exceed width
// in that case.
func Wrap(s string, width int) string {
	paragraphs := strings.Split(s, "\n")
	lines := make([]string, 0, len(paragraphs))
	for _, p := range paragraphs {
		if p == "" {
			lines = append(lines, "")
			continue
		}
		var line string
		for _, word := range strings.Fields(p) {
			switch {
			case line == "":
				line = word
			case Width(line)+1+Width(word) <= width:
				line += " " + word
			default:
				lines = append(lines, line)
				line = word
			}
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
