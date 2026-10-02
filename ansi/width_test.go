package ansi

import "testing"

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello", "hello"},
		{"styled text", NewStyle().Bold().Foreground(Red).Render("hi"), "hi"},
		{"multiple codes", CSI + "1m" + "a" + CSI + "0m" + CSI + "4m" + "b" + Reset, "ab"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripANSI(tt.input); got != tt.want {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"plain", "hello", 5},
		{"empty", "", 0},
		{"styled text has same width as plain", NewStyle().Bold().Foreground(BrightGreen).Render("hello"), 5},
		{"unicode rune count", "héllo", 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Width(tt.input); got != tt.want {
				t.Errorf("Width(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestWidthWideAndZeroWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"CJK ideographs", "你好", 4},
		{"hiragana and katakana", "ひらカナ", 8},
		{"hangul syllables", "한글", 4},
		{"fullwidth latin", "ＡＢ", 4},
		{"mixed ascii and CJK", "a你b", 4},
		{"emoji presentation", "😀🚀", 4},
		{"combining acute", "é", 1},
		{"zero width space and ZWJ", "a\u200bb\u200dc", 3},
		{"variation selector", "x️", 1},
		{"styled CJK", NewStyle().Bold().Render("你好"), 4},
		{"latin-1 unchanged", "éñü", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Width(tt.input); got != tt.want {
				t.Errorf("Width(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// #70: a two-byte ESC sequence has no width; the byte after ESC is part of
// it, not text.
func TestWidthSkipsTwoByteAndNFEscapeSequences(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want int
	}{
		{"a\x1b7b", 2},
		{"a\x1b8b", 2},
		{"a\x1b=b\x1b>c", 3},
		{"a\x1bcb", 2},
		{"a\x1b(Bb", 2},
		{"a\x1b#8b", 2},
		{"a\x1b(!Bb", 2},
		{"\x1b7", 0},
		{"a\x1b", 1},      // lone ESC at the end
		{"a\x1b(", 1},     // nF sequence with no final byte
		{"a\x1b\x01b", 2}, // ESC before a control byte is not a sequence
		{"\x1b[31mab\x1b[0m", 2},
	} {
		if got := Width(tt.in); got != tt.want {
			t.Errorf("Width(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
	if got := StripANSI("a\x1b7b\x1b(Bc"); got != "abc" {
		t.Errorf("StripANSI = %q, want abc", got)
	}
}
