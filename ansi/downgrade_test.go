package ansi

import (
	"strings"
	"testing"
)

func TestDowngradeStringRewritesColours(t *testing.T) {
	cases := []struct {
		name string
		in   string
		p    Profile
		want string
	}{
		{"rgb fg to 256", "\x1b[38;2;255;0;0mred\x1b[0m", ANSI256, "\x1b[38;5;196mred\x1b[0m"},
		{"rgb bg to 256", "\x1b[48;2;0;0;255mx", ANSI256, "\x1b[48;5;21mx"},
		{"rgb fg to 16", "\x1b[38;2;255;0;0mred", ANSI16, "\x1b[91mred"},
		{"rgb bg to 16", "\x1b[48;2;255;0;0mred", ANSI16, "\x1b[101mred"},
		{"256 fg to 16", "\x1b[38;5;196mr", ANSI16, "\x1b[91mr"},
		{"256 low index to 16", "\x1b[38;5;4mb", ANSI16, "\x1b[34mb"},
		{"256 stays at 256", "\x1b[38;5;196mr", ANSI256, "\x1b[38;5;196mr"},
		{"basic stays", "\x1b[31mr\x1b[0m", ANSI16, "\x1b[31mr\x1b[0m"},
		{"attributes kept with colour rewritten", "\x1b[1;38;2;255;0;0;4mx", ANSI16, "\x1b[1;91;4mx"},
		{"underline colour dropped at 16", "\x1b[4;58;2;255;0;0mx", ANSI16, "\x1b[4mx"},
		{"underline colour to 256", "\x1b[4;58;2;255;0;0mx", ANSI256, "\x1b[4;58;5;196mx"},
		{"colon form", "\x1b[38:2::255:0:0mx", ANSI16, "\x1b[91mx"},
		{"curly underline sub-param untouched", "\x1b[4:3mx", ANSI16, "\x1b[4:3mx"},
		{"bare reset kept", "a\x1b[mb", ANSI16, "a\x1b[mb"},
		{"non-SGR CSI untouched", "\x1b[2K\x1b[3A", NoColor, "\x1b[2K\x1b[3A"},
		{"OSC untouched", "\x1b]8;;http://x\x1b\\link\x1b]8;;\x1b\\", NoColor, "\x1b]8;;http://x\x1b\\link\x1b]8;;\x1b\\"},
	}
	for _, c := range cases {
		if got := DowngradeString(c.in, c.p); got != c.want {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

func TestDowngradeStringNoColorKeepsAttributes(t *testing.T) {
	cases := map[string]string{
		"\x1b[1;31mx\x1b[0m":                     "\x1b[1mx\x1b[0m",
		"\x1b[31mx":                              "x",
		"\x1b[38;2;1;2;3;48;5;9mx":               "x",
		"\x1b[7;4;2;3;94;105;39;49;59mx":         "\x1b[7;4;2;3mx",
		"\x1b[4:3;58;2;9;9;9mx":                  "\x1b[4:3mx",
		"\x1b[0m\x1b[1mbold\x1b[22m":             "\x1b[0m\x1b[1mbold\x1b[22m",
		"\x1b[38;5;1m\x1b[48;2;0;0;0mtwo\x1b[0m": "two\x1b[0m",
	}
	for in, want := range cases {
		if got := DowngradeString(in, NoColor); got != want {
			t.Errorf("NoColor(%q)\n got %q\nwant %q", in, got, want)
		}
	}
}

func TestDowngradeStringPassThrough(t *testing.T) {
	for _, in := range []string{"", "plain text", "\x1b[38;2;1;2;3mx"} {
		if got := DowngradeString(in, TrueColor); got != in {
			t.Errorf("TrueColor changed %q to %q", in, got)
		}
	}
	if got := DowngradeString("plain\ntext", NoColor); got != "plain\ntext" {
		t.Errorf("no escapes changed: %q", got)
	}
}

func TestDowngradeStringMalformedInputDoesNotPanicOrLeak(t *testing.T) {
	for _, in := range []string{"\x1b[", "\x1b[38;2;", "\x1b[38;2;255mx", "\x1b[38mx", "\x1b[38;5mx", "a\x1b[31", "\x1b[;;;mx", "\x1b[99999999999999999999mx"} {
		for _, p := range []Profile{NoColor, ANSI16, ANSI256} {
			_ = DowngradeString(in, p) // must not panic
		}
	}
	// A truncated colour spec at the end must not swallow following text.
	if got := DowngradeString("\x1b[38;2;255mafter", ANSI16); got == "" {
		t.Errorf("text lost")
	}
}

func FuzzDowngradeString(f *testing.F) {
	f.Add("\x1b[1;38;2;255;0;0mx\x1b[0m", uint8(1))
	f.Add("\x1b[38:2::1:2:3;4:3mx", uint8(0))
	f.Fuzz(func(t *testing.T, s string, p uint8) {
		prof := Profile(p % 4)
		out := DowngradeString(s, prof)
		// A stray ESC that is not part of a CSI is invisible on a terminal
		// (StripANSI leaves the byte in place), so compare without them.
		visible := func(x string) string { return strings.ReplaceAll(StripANSI(x), "\x1b", "") }
		if visible(out) != visible(s) {
			t.Fatalf("visible text changed: %q -> %q (profile %d)", s, out, prof)
		}
	})
}
