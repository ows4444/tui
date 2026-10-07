package ansi

import (
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "hello é 世", "hello é 世"},
		{"keeps newline", "a\nb", "a\nb"},
		{"drops tab cr nul del", "a\tb\rc\x00d\x7fe", "abcde"},
		{"drops C1", "a\u0085b\u009bc", "abc"},
		{"drops sgr", "\x1b[31mred\x1b[0m", "red"},
		{"drops csi", "a\x1b[2Jb\x1b[?25lc", "abc"},
		{"drops osc52 bel", "a\x1b]52;c;Zm9v\x07b", "ab"},
		{"drops osc8 st", "\x1b]8;;http://x\x1b\\link\x1b]8;;\x1b\\", "link"},
		{"drops dcs apc pm sos", "\x1bPq..\x1b\\a\x1b_x\x1b\\b\x1b^y\x07c\x1bXz\x07d", "abcd"},
		{"lone esc before a control", "a\x1b\x01b", "ab"},
		{"lone esc at the end", "a\x1b", "a"},
		{"two-byte esc", "a\x1b7b\x1b8c\x1b=d", "abcd"},
		{"esc plus a letter is a two-byte sequence", "a\x1bb", "a"},
		{"nF charset designation", "a\x1b(Bb\x1b#8c", "abc"},
		{"unterminated osc", "a\x1b]52;c;xx", "a"},
		{"invalid utf8 kept", "a\xffb", "a\xffb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Sanitize(tt.in); got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeKeepSGR(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"keeps sgr", "\x1b[1;31mx\x1b[0m", "\x1b[1;31mx\x1b[0m"},
		{"keeps colon sgr", "\x1b[38:2::1:2:3mx", "\x1b[38:2::1:2:3mx"},
		{"keeps bare reset", "\x1b[mx", "\x1b[mx"},
		{"drops non-sgr csi", "\x1b[31m\x1b[2J\x1b[Hx", "\x1b[31mx"},
		{"drops private m", "\x1b[?1mx", "x"},
		{"drops osc dcs apc", "\x1b]52;c;a\x07\x1bPq\x1b\\\x1b_a\x1b\\x", "x"},
		{"drops controls", "a\tb\x00\u009b", "ab"},
		{"unterminated", "x\x1b[31", "x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeKeepSGR(tt.in); got != tt.want {
				t.Errorf("SanitizeKeepSGR(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExpandTabs(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"none", "abc", "abc"},
		{"leading", "\tx", "        x"},
		{"mid", "ab\tc", "ab      c"},
		{"at stop", "12345678\tx", "12345678        x"},
		{"double", "a\t\tb", "a               b"},
		{"newline resets", "abc\n\tx", "abc\n        x"},
		{"wide rune", "世\tx", "世      x"},
		{"ansi zero width", "\x1b[31mab\x1b[0m\tx", "\x1b[31mab\x1b[0m      x"},
		{"osc zero width", "\x1b]8;;http://a\x07ab\x1b]8;;\x07\tx", "\x1b]8;;http://a\x07ab\x1b]8;;\x07      x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandTabs(tt.in)
			if got != tt.want {
				t.Errorf("ExpandTabs(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
	if w := Width(ExpandTabs("a\tb")); w != 9 {
		t.Errorf("Width(ExpandTabs(a\\tb)) = %d, want 9", w)
	}
}

func TestControlsMeasureZero(t *testing.T) {
	for _, s := range []string{"\t", "\x00", "\x01", "\x7f", "\u0080", "\u009f", "\n", "\r"} {
		if w := Width(s); w != 0 {
			t.Errorf("Width(%q) = %d, want 0", s, w)
		}
	}
	if w := Width("a\tb"); w != 2 {
		t.Errorf("Width(a\\tb) = %d, want 2", w)
	}
	if w := Width("\u00ad"); w != 1 {
		t.Errorf("soft hyphen Width = %d, want 1", w)
	}
}

func TestPlainASCIIWidthAgreesWithWidthAndSanitize(t *testing.T) {
	cases := []string{
		"", "a", "hello world", strings.Repeat("x", 7), strings.Repeat("x", 8), strings.Repeat("ab cd ", 50),
		"tab\there", "esc\x1b[1mbold", "del\x7f", "nl\nx", "cr\rx", "café", "中文", "c1\u0085x",
		strings.Repeat("x", 8) + "\t", strings.Repeat("x", 15) + "\x7f", "\x80" + strings.Repeat("x", 9),
		strings.Repeat("x", 9) + "\x1b",
	}
	for _, s := range cases {
		w, ok := PlainASCIIWidth(s)
		want := true
		for i := 0; i < len(s); i++ {
			if c := s[i]; c < 0x20 || c >= 0x7f {
				want = false
			}
		}
		if ok != want {
			t.Errorf("PlainASCIIWidth(%q) ok=%v, want %v", s, ok, want)
			continue
		}
		if ok && (w != Width(s) || Sanitize(s) != s || SanitizeKeepSGR(s) != s || ExpandTabs(s) != s) {
			t.Errorf("PlainASCIIWidth(%q) = %d but Width=%d or a sanitizer changed it", s, w, Width(s))
		}
		if !ok && w != 0 {
			t.Errorf("PlainASCIIWidth(%q) = %d with ok false, want 0", s, w)
		}
	}
}

func FuzzPlainASCIIWidth(f *testing.F) {
	for _, s := range []string{"", "hello world, this is text", "a\tb", "x\x7fy", "café au lait plus text"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		w, ok := PlainASCIIWidth(s)
		plain := true
		for i := 0; i < len(s); i++ {
			if s[i] < 0x20 || s[i] >= 0x7f {
				plain = false
			}
		}
		if ok != plain || (ok && w != len(s)) {
			t.Fatalf("PlainASCIIWidth(%q) = %d, %v", s, w, ok)
		}
	})
}
