package ansi

import "testing"

func TestTruncateStartAndMiddle(t *testing.T) {
	for _, tt := range []struct {
		name, got, want string
	}{
		{"start", TruncateStart("/usr/local/bin/tool", 10, "~"), "~/bin/tool"},
		{"start fits", TruncateStart("short", 10, "~"), "short"},
		{"start no glyph", TruncateStart("abcdefgh", 3, ""), "fgh"},
		{"start glyph too wide", TruncateStart("abcdefgh", 2, "..."), ".."},
		{"middle", TruncateMiddle("/usr/local/bin/tool", 11, "~"), "/usr/~/tool"},
		{"middle odd room", TruncateMiddle("abcdefghij", 6, "~"), "abc~ij"},
		{"middle even room", TruncateMiddle("abcdefghij", 7, "~"), "abc~hij"},
		{"middle fits", TruncateMiddle("abc", 5, "~"), "abc"},
		{"middle no glyph", TruncateMiddle("abcdefgh", 4, ""), "abgh"},
	} {
		if tt.got != tt.want {
			t.Errorf("%s = %q; want %q", tt.name, tt.got, tt.want)
		}
	}
	if TruncateStart("abc", 0, "~") != "" || TruncateMiddle("abc", -1, "~") != "" {
		t.Error("a zero width must give nothing")
	}
}

func TestTruncateKeepsStyleAndWideRunes(t *testing.T) {
	red := NewStyle().Foreground(Red).Render("abcdefgh")
	got := TruncateStart(red, 4, "~")
	if StripANSI(got) != "~fgh" || Width(got) != 4 {
		t.Fatalf("styled start: %q", got)
	}
	if !contains(got, "\x1b[31m") {
		t.Fatalf("the colour was lost at the cut: %q", got)
	}
	wide := "世界世界世界" // 12 columns
	for _, w := range []int{5, 6, 7} {
		if g := TruncateStart(wide, w, "~"); Width(g) > w {
			t.Errorf("TruncateStart width %d gave %d columns: %q", w, Width(g), g)
		}
		if g := TruncateMiddle(wide, w, "~"); Width(g) > w {
			t.Errorf("TruncateMiddle width %d gave %d columns: %q", w, Width(g), g)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestJustify(t *testing.T) {
	for _, tt := range []struct {
		in    string
		width int
		want  string
	}{
		{"a b c", 9, "a   b   c"},
		{"a b c", 8, "a   b  c"},
		{"one two", 10, "one    two"},
		{"nogaps", 12, "nogaps"},
		{"already wide", 5, "already wide"},
		{" lead", 9, " lead"},
		{"trail ", 9, "trail "},
		{"", 9, ""},
	} {
		if got := Justify(tt.in, tt.width); got != tt.want {
			t.Errorf("Justify(%q, %d) = %q; want %q", tt.in, tt.width, got, tt.want)
		}
	}
	styled := NewStyle().Bold().Render("ab") + " " + NewStyle().Italic().Render("cd")
	got := Justify(styled, 8)
	if Width(got) != 8 || StripANSI(got) != "ab    cd" {
		t.Fatalf("styled: %q", got)
	}
	link := "\x1b]8;;http://x y\x1b\\ab\x1b]8;;\x1b\\ cd"
	g := Justify(link, 6)
	if Width(g) != 6 || !contains(g, "http://x y") {
		t.Fatalf("a space inside an escape sequence was widened, or the line is not 6 wide: %q", g)
	}
}
