package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// osc8Balanced reports whether every OSC 8 open in s is closed and none is
// closed without being open. Copied from ansi/osc_test.go (which is not
// exported) and made self-contained by scanning for the sequence directly.
func osc8Balanced(s string) bool {
	const intro = "\x1b]8;"
	depth := 0
	for {
		i := strings.Index(s, intro)
		if i < 0 {
			return depth == 0
		}
		s = s[i+len(intro):]
		end := len(s)
		termLen := 0
		if j := strings.IndexByte(s, 0x07); j >= 0 {
			end, termLen = j, 1
		}
		if j := strings.Index(s, "\x1b\\"); j >= 0 && j < end {
			end, termLen = j, 2
		}
		if termLen == 0 {
			return false // unterminated OSC
		}
		body := s[:end] // params;uri
		s = s[end+termLen:]
		uri := body[strings.IndexByte(body, ';')+1:]
		if uri == "" {
			depth--
		} else {
			depth++
		}
		if depth < 0 || depth > 1 {
			return false
		}
	}
}

func TestLinkInBoxKeepsOSC8Balanced(t *testing.T) { // criterion 4
	tt := theme.DarkTheme()
	link := Link("click here for the docs", "https://example.com/a/very/long/url", true, tt)
	for _, w := range []int{0, 12, 20, 40, 80} {
		got := Box("Title", link+"\nplain", tt, w)
		if !osc8Balanced(got) {
			t.Errorf("Box(width=%d) unbalanced OSC 8: %q", w, got)
		}
		// Per-row balance only holds while the link is not wrapped across
		// rows; a wrapped link is open on one row and closed on the next.
		if w == 0 || w >= 40 {
			for i, l := range strings.Split(got, "\n") {
				if !osc8Balanced(l) {
					t.Errorf("Box(width=%d) line %d unbalanced: %q", w, i, l)
				}
			}
		}
	}
}

func TestLinkInTableKeepsOSC8Balanced(t *testing.T) { // criterion 4
	tt := theme.DarkTheme()
	link := Link("docs", "https://example.com/some/path", false, tt)
	long := Link("a much longer link label", "https://example.com/other", true, tt)
	got := Table([]string{"Name", "Link"}, [][]string{{"one", link}, {"two", long}, {"three", "plain"}}, tt)
	if !osc8Balanced(got) {
		t.Errorf("Table unbalanced OSC 8: %q", got)
	}
	for i, l := range strings.Split(got, "\n") {
		if !osc8Balanced(l) {
			t.Errorf("Table line %d unbalanced: %q", i, l)
		}
	}
	// Width is measured on visible text, so the URL must not widen columns.
	if strings.Contains(ansi.StripANSI(got), "\x1b") || strings.Contains(ansi.StripANSI(got), "]8;") {
		t.Errorf("StripANSI left escape residue: %q", ansi.StripANSI(got))
	}
}
