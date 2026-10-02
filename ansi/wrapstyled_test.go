package ansi

import (
	"strings"
	"testing"
)

func TestWrapStyledLinesSelfContained(t *testing.T) {
	red := NewStyle().Foreground(Red)
	got := WrapStyled(red.Render("alpha beta gamma delta epsilon zeta eta theta"), 20)
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected wrapping, got %q", got)
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, "\x1b[31m") || !strings.HasSuffix(l, Reset) {
			t.Errorf("line not self-contained: %q", l)
		}
		if Width(l) > 20 {
			t.Errorf("line too wide: %q", l)
		}
	}
	if StripANSI(got) != "alpha beta gamma\ndelta epsilon zeta\neta theta" {
		t.Errorf("text = %q", StripANSI(got))
	}
}

func TestWrapStyledStopsAtReset(t *testing.T) {
	in := "\x1b[1mbold one\x1b[0m plain words here"
	got := WrapStyled(in, 10)
	for i, l := range strings.Split(got, "\n") {
		if i > 0 && strings.Contains(l, "\x1b[1m") {
			t.Errorf("bold leaked to line %d: %q", i, l)
		}
	}
}

func TestWrapStyledLink(t *testing.T) {
	got := WrapStyled(Hyperlink("one two three four", "http://x"), 9)
	for _, l := range strings.Split(got, "\n") {
		if !strings.HasPrefix(l, "\x1b]8;;http://x\x1b\\") || !strings.HasSuffix(l, linkClose) {
			t.Errorf("link not self-contained: %q", l)
		}
	}
}

func TestWrapStyledPlainMatchesWrap(t *testing.T) {
	for _, in := range []string{"", "a", "hello world foo bar baz", "a\n\nb c d e f", "averyveryverylongword x"} {
		for _, w := range []int{1, 5, 10, 40} {
			if a, b := Wrap(in, w), WrapStyled(in, w); a != b {
				t.Errorf("Wrap(%q,%d)=%q WrapStyled=%q", in, w, a, b)
			}
		}
	}
}

func TestWrapStyledLargeParagraph(t *testing.T) {
	in := NewStyle().Bold().Render(strings.Repeat("lorem ipsum ", 5000))
	out := WrapStyled(in, 60)
	if got := len(strings.Fields(StripANSI(out))); got != 10000 {
		t.Fatalf("words = %d", got)
	}
}

func BenchmarkWrapStyled10kWords(b *testing.B) {
	in := NewStyle().Bold().Render(strings.Repeat("lorem ipsum ", 5000))
	b.SetBytes(int64(len(in)))
	for i := 0; i < b.N; i++ {
		WrapStyled(in, 60)
	}
}

func TestSpanNestedRestoresOuter(t *testing.T) {
	outer := NewStyle().Foreground(Red)
	inner := NewStyle().Bold()
	got := outer.Span("a ", inner.Render("b"), " c")
	want := "\x1b[31ma \x1b[1mb\x1b[0m\x1b[31m c\x1b[0m"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
	if end := outer.Span(inner.Render("b")); strings.HasSuffix(end, "\x1b[31m\x1b[0m") {
		t.Errorf("redundant re-open at end: %q", end)
	}
}
