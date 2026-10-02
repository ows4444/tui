package markdown

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func maxWidth(s string) int {
	w := 0
	for _, l := range strings.Split(s, "\n") {
		w = max(w, ansi.Width(l))
	}
	return w
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// Criterion #77: a pipe table with a delimiter row is drawn as aligned
// columns, honouring :--, :-: and --:.
func TestTableAlignment(t *testing.T) {
	md := "| L | C | R |\n|:--|:-:|--:|\n| a | b | c |\n| aaaa | bbbb | cccc |"
	th := theme.DarkTheme().ASCII()
	got := ansi.StripANSI(Render(md, 40, th))
	want := strings.Join([]string{
		"L    |  C   |    R",
		"------------------",
		"a    |  b   |    c",
		"aaaa | bbbb | cccc",
	}, "\n")
	if got != want {
		t.Errorf("table:\n%s\nwant:\n%s", got, want)
	}
	// Without a delimiter row it is only a paragraph.
	if p := plain("| a | b |\n| c | d |", 40); !strings.HasPrefix(p, "| a | b |") {
		t.Errorf("no delimiter row rendered as %q", p)
	}
	// A mismatched delimiter row is not a table either.
	if got := plain("a | b\n--|--|--\nx | y", 40); strings.Contains(got, "-------") {
		t.Errorf("column-count mismatch drew a table: %q", got)
	}
}

// Criterion #77: rows with too few cells are padded and extras dropped,
// and a table ends at a blank line.
func TestTableRaggedRows(t *testing.T) {
	got := plain("a | b\n- | -\n1\n1 | 2 | 3\n\nafter", 30)
	want := "a │ b\n─────\n1 │\n1 │ 2\n\nafter"
	if got != want {
		t.Errorf("ragged table = %q, want %q", got, want)
	}
}

// Criterion #78: a table wider than the width is wrapped or truncated so
// no row exceeds it.
func TestTableNarrow(t *testing.T) {
	md := "| name | description |\n|---|---|\n| alpha | a very long description that will not fit |\n| b | " +
		strings.Repeat("x", 80) + " |"
	for w := 1; w <= 60; w++ {
		got := ansi.StripANSI(Render(md, w, theme.DarkTheme().ASCII()))
		if mw := maxWidth(got); mw > w {
			t.Fatalf("width %d: row is %d columns:\n%s", w, mw, got)
		}
	}
	got := plain(md, 30)
	if !strings.Contains(got, "alpha") || !strings.Contains(got, "description") {
		t.Errorf("narrow table lost content:\n%s", got)
	}
	// Tables also fit inside quotes and lists.
	got = plain("> "+strings.ReplaceAll(md, "\n", "\n> "), 30)
	if mw := maxWidth(got); mw > 30 {
		t.Errorf("quoted table is %d wide:\n%s", mw, got)
	}
}

// Criterion #79: a text line followed by = or - is a level 1 or 2 heading.
func TestSetextHeading(t *testing.T) {
	cases := []struct {
		md   string
		want string
	}{
		{"Title\n=====", "H1(Title)"},
		{"Title\n---", "H2(Title)"},
		{"Title\n=", "H1(Title)"},
		{"Two\nlines\n===", "H1(Two\nlines)"},
		{"para\n\nSub\n---\ntext", "P(para) H2(Sub) P(text)"},
		{"a\n--- \nb", "H2(a) P(b)"},
		{"---\n", "HR"},
		{"a\n\n---", "P(a) HR"},
		{"a\n== =", "P(a\n== =)"},
	}
	for _, c := range cases {
		if got := dump(parseBlocks(splitLines(c.md), 0, 0)); got != c.want {
			t.Errorf("%q = %s, want %s", c.md, got, c.want)
		}
	}
	// Level 1 is underlined like an ATX h1, and the underline is consumed.
	if got, want := Render("Title\n===", 20, dark), Render("# Title", 20, dark); got != want {
		t.Errorf("setext h1 = %q, want %q", got, want)
	}
	if got, want := Render("Title\n---", 20, dark), Render("## Title", 20, dark); got != want {
		t.Errorf("setext h2 = %q, want %q", got, want)
	}
}

// upperLexer colours every line in the Keyword class for lang "toy".
type upperLexer struct{ calls int }

func (l *upperLexer) Lex(code, lang string) ([][]Span, bool) {
	l.calls++
	if lang != "toy" {
		return nil, false
	}
	var out [][]Span
	for _, ln := range strings.Split(code, "\n") {
		out = append(out, []Span{{Text: ln, Class: Keyword}})
	}
	return out, true
}

// Criterion #80: a supplied Lexer colours the fence for its language.
func TestLexerColoursFence(t *testing.T) {
	md := "```toy\nhello\nworld\n```"
	lx := &upperLexer{}
	got := RenderWith(md, 20, dark, Options{Lexer: lx})
	if lx.calls == 0 {
		t.Fatal("lexer not called")
	}
	kw := ansi.NewStyle().Foreground(dark.Primary).Render("hello")
	if !strings.Contains(got, kw) {
		t.Errorf("fence not coloured by lexer: %q", got)
	}
	if plain, want := ansi.StripANSI(got), codeBox("hello\nworld", 20); plain != want {
		t.Errorf("lexed box differs from the plain box:\n%s\nwant:\n%s", plain, want)
	}
	if mw := maxWidth(ansi.StripANSI(RenderWith(md, 8, dark, Options{Lexer: lx}))); mw != 8 {
		t.Errorf("narrow lexed box is %d wide, want 8", mw)
	}
	for w := 1; w < 8; w++ {
		if mw := maxWidth(ansi.StripANSI(RenderWith(md, w, dark, Options{Lexer: lx}))); mw > w {
			t.Errorf("width %d: lexed box is %d wide", w, mw)
		}
	}
	// Control bytes in a lexer's output cannot reach the terminal.
	bad := lexerFunc(func(code, lang string) ([][]Span, bool) {
		return [][]Span{{{Text: "a\x1b[31mb\nc", Class: String}}}, true
	})
	if got := RenderWith("```x\nq\n```", 20, dark, Options{Lexer: bad}); strings.Contains(got, "\x1b[31m") {
		t.Errorf("escape sequence from a lexer survived: %q", got)
	}
}

type lexerFunc func(code, lang string) ([][]Span, bool)

func (f lexerFunc) Lex(code, lang string) ([][]Span, bool) { return f(code, lang) }

// Criterion #80: with no Lexer, or one that declines the language, fences
// are drawn exactly as before (plain), and DefaultLexer offers the built-in
// languages to callers who want them.
func TestNoLexerIsUnchanged(t *testing.T) {
	for _, md := range []string{"```go\nfunc main() { return 1 }\n```", "~~~ go title=x\nx := 1\n~~~", "```nope\nx\n```"} {
		want := Render(md, 40, dark)
		if got := RenderWith(md, 40, dark, Options{}); got != want {
			t.Errorf("Options{} differs from Render:\n%q\n%q", got, want)
		}
		if got := RenderWith(md, 40, dark, Options{Lexer: &upperLexer{}}); got != want {
			t.Errorf("declining lexer changed %q:\n%q\n%q", md, got, want)
		}
	}
	if got, want := plain("```go\nx := 1\n```", 20), codeBox("x := 1", 20); got != want {
		t.Errorf("plain go fence = %q, want %q", got, want)
	}
	// The info string's first word is the language, for either fence.
	lx := &upperLexer{}
	for _, md := range []string{"~~~ toy title=x\nq\n~~~", "```toy\nq\n```"} {
		kw := ansi.NewStyle().Foreground(dark.Primary).Render("q")
		if got := RenderWith(md, 20, dark, Options{Lexer: lx}); !strings.Contains(got, kw) {
			t.Errorf("lang of %q not passed to the lexer: %q", md, got)
		}
	}
	// DefaultLexer adapts internal/highlight: it colours Go and declines the rest.
	lines, ok := DefaultLexer().Lex("func x() {}", "go")
	if !ok || len(lines) != 1 || lines[0][0] != (Span{Text: "func", Class: Keyword}) {
		t.Errorf("DefaultLexer = %v, %v", lines, ok)
	}
	if _, ok := DefaultLexer().Lex("x", "nope"); ok {
		t.Error("DefaultLexer accepted an unknown language")
	}
	kw := ansi.NewStyle().Foreground(dark.Primary).Render("func")
	if got := RenderWith("```go\nfunc\n```", 20, dark, Options{Lexer: DefaultLexer()}); !strings.Contains(got, kw) {
		t.Errorf("DefaultLexer fence not coloured: %q", got)
	}
}

// Criterion #81: output stays 7-bit ASCII under theme.Dark.ASCII().
func TestASCIIOutput(t *testing.T) {
	th := theme.DarkTheme().ASCII()
	md := "Title\n=====\n\nSub\n---\n\n| a | b |\n|:-|-:|\n| 1 | 2 |\n\n" +
		"```toy\nx\n```\n\n```go\nfunc() {}\n```\n\n- item\n  - nested\n\n> quote\n\n---"
	for _, w := range []int{4, 12, 40} {
		for _, o := range []Options{{}, {Lexer: &upperLexer{}}} {
			got := RenderWith(md, w, th, o)
			if !isASCII(ansi.StripANSI(got)) {
				t.Errorf("width %d: non-ASCII output:\n%s", w, ansi.StripANSI(got))
			}
		}
	}
}
