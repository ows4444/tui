package ansi

import "testing"

// FuzzStripANSI feeds arbitrary strings into StripANSI, which scans for the
// CSI introducer byte-by-byte and must never panic on truncated, malformed,
// or adversarial escape sequences — it runs on every render diff via
// Width, so a crash here is a crash on untrusted View() output.
func FuzzStripANSI(f *testing.F) {
	seeds := []string{
		"",
		"\x1b",
		"\x1b[",
		"\x1b[m",
		"\x1b[1;31mred\x1b[0m",
		"\x1b[999999999999999999999m",
		"plain text, no escapes",
		"\x1b[38;2;255;0;0m\x1b[48;5;12mtext\x1b[0m",
		"\x1b[",
		"[not-actually-csi",
		"hello, 世界 🎉",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		// Must not panic. StripANSI's result also must never grow longer
		// than the input (it only removes bytes, never adds any).
		if got := StripANSI(s); len(got) > len(s) {
			t.Fatalf("StripANSI(%q) = %q, longer than input", s, got)
		}
	})
}

func FuzzTextWrap(f *testing.F) {
	for _, s := range []string{"", "hello world", "a  b\n\nc", "世界 wide", "é 👨‍👩‍👧 x", "\x1b[31mred\x1b[0m"} {
		f.Add(s, 3)
	}
	f.Fuzz(func(t *testing.T, s string, w int) {
		w = (w%20+20)%20 + 1
		text := NewText(s, NewStyle().Bold())
		for _, l := range text.Wrap(w) {
			if l.Width() > w && l.Width() > 2 {
				t.Fatalf("line %q is %d wide, over %d", l.Plain(), l.Width(), w)
			}
			_ = l.String()
		}
		_ = text.Truncate(w).String()
		_ = text.Slice(w/2, w).Width()
	})
}
