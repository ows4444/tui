package render

import "testing"

// An inline image ends at its BEL or its ST; one that never ends, or holds a
// stray ESC, is refused with a reason.
func TestParseInlineImage(t *testing.T) {
	const open = inlineImageOSC + "inline=1:AAAA"
	for line, want := range map[string]int{
		open + "\a":         len(open) + 1,
		open + "\x1b\\":     len(open) + 2,
		open + "\arest":     len(open) + 1,
		open + "\x1b\\rest": len(open) + 2,
	} {
		c := &Cells{}
		if end, ok := c.parseInlineImage(line, 0); !ok || end != want {
			t.Errorf("parseInlineImage(%q) = %d, %v, want %d", line, end, ok, want)
		}
	}
	for line, reason := range map[string]string{
		open:             "unterminated_OSC",
		open + "\x1b[0m": "malformed_string_sequence",
		open + "\x1b":    "malformed_string_sequence",
	} {
		c := &Cells{}
		if _, ok := c.parseInlineImage(line, 0); ok || c.reason != reason {
			t.Errorf("parseInlineImage(%q): ok=%v reason %q, want %q", line, ok, c.reason, reason)
		}
	}
}
