package ansi

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func spansOf(t Text) []Span { return []Span(t) }

func TestMarkupStylesAndNesting(t *testing.T) {
	bold := NewStyle().Bold()
	boldRed := bold.Foreground(Red)
	got := Markup("a [bold]b [red]c[/] d[/] e")
	want := []Span{
		{"a ", Style{}}, {"b ", bold}, {"c", boldRed}, {" d", bold}, {" e", Style{}},
	}
	if g := spansOf(got); len(g) != len(want) {
		t.Fatalf("spans = %+v", g)
	} else {
		for i := range want {
			if g[i] != want[i] {
				t.Errorf("span %d = %+v, want %+v", i, g[i], want[i])
			}
		}
	}
}

func TestMarkupWordsAndColours(t *testing.T) {
	cases := map[string]Style{
		"[bold]x[/]":             NewStyle().Bold(),
		"[B]x[/]":                NewStyle().Bold(),
		"[dim italic]x[/]":       NewStyle().Faint().Italic(),
		"[underline]x[/]":        NewStyle().Underline(),
		"[blink reverse]x[/]":    NewStyle().Blink().Reverse(),
		"[strike conceal]x[/]":   NewStyle().Strikethrough().Conceal(),
		"[overline]x[/]":         NewStyle().Overline(),
		"[red]x[/]":              NewStyle().Foreground(Red),
		"[bright_cyan]x[/]":      NewStyle().Foreground(BrightCyan),
		"[bright-cyan]x[/]":      NewStyle().Foreground(BrightCyan),
		"[RED]x[/]":              NewStyle().Foreground(Red),
		"[#f80]x[/]":             NewStyle().Foreground(RGB{R: 0xff, G: 0x88, B: 0}),
		"[#1a2B3c]x[/]":          NewStyle().Foreground(RGB{R: 0x1a, G: 0x2b, B: 0x3c}),
		"[white on blue]x[/]":    NewStyle().Foreground(White).Background(Blue),
		"[on #000000]x[/]":       NewStyle().Background(RGB{}),
		"[link=https://a.b]x[/]": NewStyle().Link("https://a.b"),
	}
	for in, want := range cases {
		got, err := ParseMarkup(in)
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if len(got) != 1 || got[0].Text != "x" || got[0].Style != want {
			t.Errorf("%q = %+v, want style %+v", in, got, want)
		}
	}
}

func TestMarkupClosing(t *testing.T) {
	// "[/word]" closes the last open tag with exactly those words, and the ones above it.
	got := Markup("[bold]a[red]b[/bold]c")
	if g := spansOf(got); len(g) != 3 || g[2].Style != (Style{}) || g[2].Text != "c" {
		t.Errorf("[/bold] did not close everything above bold: %+v", g)
	}
	// Tags still open at the end close themselves.
	if g := spansOf(Markup("[bold]a")); len(g) != 1 || g[0].Style != NewStyle().Bold() {
		t.Errorf("unclosed tag: %+v", g)
	}
	// Words are compared as written, spaces collapsed.
	if g := spansOf(Markup("[bold   red]a[/bold red]b")); len(g) != 2 || g[1].Style != (Style{}) {
		t.Errorf("close with spaces: %+v", g)
	}
}

func TestMarkupNonTagBracketsAreText(t *testing.T) {
	for _, in := range []string{"a [1] b", "a [ ] b", "[] x", "x [", "x ] y", "[!] z", "[-bold]"} {
		got, err := ParseMarkup(in)
		if err != nil || got.Plain() != in {
			t.Errorf("%q = %q, %v; want the same text and no error", in, got.Plain(), err)
		}
	}
}

func TestParseMarkupErrorsAndMarkupKeepsLiteral(t *testing.T) {
	for _, in := range []string{"[Note] hi", "[bold nonsense]x", "[on]x", "[on nothing]x", "x[/]", "[/bold]x", "[#12]x", "[#gg0000]x"} {
		if _, err := ParseMarkup(in); err == nil {
			t.Errorf("ParseMarkup(%q) did not fail", in)
		}
		if got := Markup(in).Plain(); got != in {
			t.Errorf("Markup(%q).Plain() = %q, want the text as written", in, got)
		}
	}
	// ParseMarkup returns the text before the error.
	got, err := ParseMarkup("ok [red]fine[/] then [bad]")
	if err == nil || got.Plain() != "ok fine then " {
		t.Errorf("got %q, %v", got.Plain(), err)
	}
}

func TestMarkupEscape(t *testing.T) {
	if got := Markup(`\[bold]x`).Plain(); got != "[bold]x" {
		t.Errorf(`\[ = %q`, got)
	}
	if got := Markup(`a \ b \\ c`).Plain(); got != `a \ b \\ c` {
		t.Errorf("lone backslashes changed: %q", got)
	}
	if got := EscapeMarkup("[red]x[/]"); got != `\[red]x\[/]` {
		t.Errorf("EscapeMarkup = %q", got)
	}
}

func TestEscapeMarkupRoundTrips(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	alphabet := []string{"[", "]", `\`, "a", "b", "/", "#", " ", "bold", "red", "=", "é", "日"}
	for n := 0; n < 3000; n++ {
		var b strings.Builder
		for i, k := 0, r.Intn(12); i < k; i++ {
			b.WriteString(alphabet[r.Intn(len(alphabet))])
		}
		s := b.String()
		got, err := ParseMarkup(EscapeMarkup(s))
		if err != nil || got.Plain() != s {
			t.Fatalf("round trip of %q gave %q, %v", s, got.Plain(), err)
		}
		for _, sp := range got {
			if sp.Style != (Style{}) {
				t.Fatalf("round trip of %q is styled: %+v", s, got)
			}
		}
	}
}

func TestMarkupCannotCarryControlCodes(t *testing.T) {
	got := Markup("[red]a\x1b[2Jb\x1b]0;t\x07c[/]\x07d\t|")
	if p := got.Plain(); strings.ContainsAny(p, "\x1b\x07") || !strings.HasPrefix(p, "abcd") {
		t.Errorf("Plain = %q", p)
	}
	// A link target with a control character is dropped, the text stays.
	if g := spansOf(Markup("[link=http://a\x1b\\b]x[/]")); len(g) == 0 || g[0].Style.link != "" {
		t.Errorf("unsafe link kept: %+v", g)
	}
}

func TestMarkupJoinsNeighboursInOneStyle(t *testing.T) {
	if g := spansOf(Markup("a[bold][/]b")); len(g) != 1 || g[0].Text != "ab" {
		t.Errorf("empty tag split the span: %+v", g)
	}
}

func TestMarkupRenders(t *testing.T) {
	got := Markup("[bold]hi[/] there").String()
	if StripANSI(got) != "hi there" || !strings.Contains(got, "\x1b[") {
		t.Errorf("String = %q", got)
	}
}

func ExampleMarkup() {
	name := "notes [draft].txt" // not ours: escape it
	t := Markup("[bold red]error[/] in " + EscapeMarkup(name))
	fmt.Println(t.Plain())
	// Output: error in notes [draft].txt
}
