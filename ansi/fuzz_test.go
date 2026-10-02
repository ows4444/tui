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
