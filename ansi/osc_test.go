package ansi

import (
	"strings"
	"testing"
)

const (
	oscOpenBEL  = "\x1b]8;;https://example.com/a/very/long/url\x07"
	oscCloseBEL = "\x1b]8;;\x07"
	oscOpenST   = "\x1b]8;;https://example.com\x1b\\"
)

// balanced reports whether every OSC 8 open in s is closed.
func balanced(s string) bool {
	depth := 0
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			continue
		}
		end, ok := escEnd(s, i)
		if !ok || end <= i {
			continue
		}
		if isLink, opens := osc8(s[i:end]); isLink {
			if opens {
				depth++
			} else {
				depth--
			}
		}
		i = end - 1
	}
	return depth == 0
}

func TestWidthSkipsOSC8(t *testing.T) { // criterion 1
	for _, s := range []string{
		oscOpenBEL + "click" + oscCloseBEL,
		oscOpenST + "click" + "\x1b]8;;\x1b\\",
		"\x1b[4m" + oscOpenBEL + "click" + oscCloseBEL + "\x1b[0m",
	} {
		if got := Width(s); got != 5 {
			t.Errorf("Width(%q) = %d, want 5", s, got)
		}
	}
	if got := Width("a\x1bPdcs data\x1b\\b\x1b_apc\x07c\x1bXsos\x07\x1b^pm\x07"); got != 3 {
		t.Errorf("Width with DCS/APC/SOS/PM = %d, want 3", got)
	}
}

func TestTruncateClosesOSC8(t *testing.T) { // criterion 2
	link := oscOpenBEL + "click here" + oscCloseBEL + " tail"
	for _, w := range []int{1, 5, 10, 12, 15, 100} {
		got := Truncate(link, w)
		if !balanced(got) {
			t.Errorf("Truncate(w=%d) = %q, OSC 8 unbalanced", w, got)
		}
		if gw := Width(got); gw > w {
			t.Errorf("Truncate(w=%d) width %d", w, gw)
		}
	}
	if got := Truncate(link, 5); got != oscOpenBEL+"click"+linkClose {
		t.Errorf("Truncate mid-link = %q", got)
	}
	if got := Truncate(link, 100); got != link {
		t.Errorf("uncut string changed: %q", got)
	}
	// Cut on a cluster boundary that rolls back past the link open.
	if got := Truncate("ab"+oscOpenBEL+"éx"+oscCloseBEL, 3); !balanced(got) {
		t.Errorf("rollback unbalanced: %q", got)
	}
	// Unterminated OSC is dropped, not kept open.
	if got := Truncate("ab\x1b]8;;http", 10); got != "ab" {
		t.Errorf("unterminated OSC = %q", got)
	}
	// Style and link both open.
	got := Truncate("\x1b[1m"+oscOpenBEL+"click", 3)
	if got != "\x1b[1m"+oscOpenBEL+"cli"+Reset+linkClose {
		t.Errorf("style+link = %q", got)
	}
}

func TestStripANSIRemovesStringSequences(t *testing.T) { // criterion 3
	cases := map[string]string{
		"a" + oscOpenBEL + "b" + oscCloseBEL + "c": "abc",
		"a" + oscOpenST + "b\x1b]8;;\x1b\\c":       "abc",
		"a\x1bPq1;2\x1b\\b":                        "ab",
		"a\x1b_Gdata\x07b":                         "ab",
		"a\x1b]0;title\x07b\x1b[1mc\x1b[0m":        "abc",
		"a\x1bXs\x1b\\b\x1b^p\x07c":                "abc",
		"a\x1b]8;;unterminated":                    "a",
	}
	for in, want := range cases {
		if got := StripANSI(in); got != want {
			t.Errorf("StripANSI(%q) = %q, want %q", in, got, want)
		}
		if got := Width(in); got != Width(want) {
			t.Errorf("Width(%q) = %d, want %d", in, got, Width(want))
		}
	}
}

func TestTrimLeftWidthSkipsOSC(t *testing.T) {
	s := oscOpenBEL + "click" + oscCloseBEL + "xyz"
	if got := StripANSI(TrimLeftWidth(s, 5)); got != "xyz" {
		t.Errorf("TrimLeftWidth = %q", got)
	}
}

func TestWidthASCIIZeroAllocs(t *testing.T) { // criterion 5
	s := strings.Repeat("hello \x1b[1mworld\x1b[0m ", 8)
	l := oscOpenBEL + "link" + oscCloseBEL
	if n := testing.AllocsPerRun(100, func() { Width(s); Width(l) }); n != 0 {
		t.Errorf("Width allocs = %v, want 0", n)
	}
}
