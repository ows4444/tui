package markdown

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

var dark = theme.DarkTheme()

func plain(md string, width int) string { return ansi.StripANSI(Render(md, width, dark)) }

func codeBox(code string, width int) string {
	return ansi.StripANSI(widgets.CodeBlock(code, width, false, dark))
}

func indent(s, pad string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n")
}

func TestRenderEmptyAndWidth(t *testing.T) {
	for _, w := range []int{0, -1, -40} {
		if got := Render("# hi", w, dark); got != "" {
			t.Errorf("width %d = %q, want empty", w, got)
		}
	}
	for _, md := range []string{"", "   ", "\n\n", " \t\n \n"} {
		if got := Render(md, 20, dark); got != "" {
			t.Errorf("Render(%q) = %q, want empty", md, got)
		}
	}
}

func TestRenderBlockSeparation(t *testing.T) {
	tests := []struct{ name, md, want string }{
		{"single blank between paragraphs", "a\n\n\n\nb", "a\n\nb"},
		{"no leading or trailing blanks", "\n\na\n\n", "a"},
		{"heading then paragraph", "# a\n\nb", "a\n\nb"},
		{"tight heading paragraph still separated", "# a\nb", "a\n\nb"},
		{"empty heading leaves no gap", "a\n\n#\n\nb", "a\n\nb"},
		{"rule between", "a\n\n---\n\nb", "a\n\n─────\n\nb"},
	}
	for _, tt := range tests {
		if got := plain(tt.md, 5); got != tt.want {
			t.Errorf("%s: Render(%q, 5)\n got  %q\n want %q", tt.name, tt.md, got, tt.want)
		}
	}
}

func TestRenderParagraphs(t *testing.T) {
	tests := []struct {
		name, md string
		width    int
		want     string
	}{
		{"wraps on words", "the quick brown fox jumps", 10, "the quick\nbrown fox\njumps"},
		{"fits exactly", "abcde fghij", 11, "abcde fghij"},
		{"one over", "abcde fghij", 10, "abcde\nfghij"},
		{"soft break joins", "a\nb", 20, "a b"},
		{"hard break spaces", "a  \nb", 20, "a\nb"},
		{"hard break backslash", "a\\\nb", 20, "a\nb"},
		{"whitespace collapses", "a    b\t\tc", 20, "a b c"},
		{"two paragraphs", "a\n\nb", 20, "a\n\nb"},
		{"long word broken", "abcdefghij", 4, "abcd\nefgh\nij"},
		{"broken word then more text", "abcdefghij kl", 4, "abcd\nefgh\nij\nkl"},
		{"text before broken word", "ab cdefghij", 4, "ab\ncdef\nghij"},
		{"wide runes never split", "你好世界", 5, "你好\n世界"},
		{"crlf", "a\r\nb\r\n\r\nc", 20, "a b\n\nc"},
		{"styled pieces stay one word", "foo**bar**baz", 20, "foobarbaz"},
		{"styled word wraps whole", "aaa **bbb** ccc", 7, "aaa bbb\nccc"},
		{"hard break at start ignored", "\\\nx", 20, "x"},
	}
	for _, tt := range tests {
		if got := plain(tt.md, tt.width); got != tt.want {
			t.Errorf("%s: Render(%q, %d)\n got  %q\n want %q", tt.name, tt.md, tt.width, got, tt.want)
		}
	}
}

func TestRenderHeadings(t *testing.T) {
	th := dark
	h1 := ansi.NewStyle().Foreground(th.Primary).Bold().Underline()
	h2 := ansi.NewStyle().Foreground(th.Primary).Bold()
	h3 := ansi.NewStyle().Foreground(th.Text).Bold()

	if got, want := Render("# Title", 40, th), h1.Render("Title"); got != want {
		t.Errorf("h1 = %q, want %q", got, want)
	}
	if got, want := Render("## Title", 40, th), h2.Render("Title"); got != want {
		t.Errorf("h2 = %q, want %q", got, want)
	}
	for _, md := range []string{"### T", "#### T", "##### T", "###### T"} {
		if got, want := Render(md, 40, th), h3.Render("T"); got != want {
			t.Errorf("%q = %q, want %q", md, got, want)
		}
	}
	// Multi-word headings style per word, with unstyled spaces between.
	if got, want := Render("## a b", 40, th), h2.Render("a")+" "+h2.Render("b"); got != want {
		t.Errorf("multi-word h2 = %q, want %q", got, want)
	}
	// Inline formatting inside a heading; wrapping.
	if got, want := Render("## *a*", 40, th), ansi.NewStyle().Foreground(th.Primary).Bold().Italic().Render("a"); got != want {
		t.Errorf("emphasis in heading = %q, want %q", got, want)
	}
	if got, want := plain("## a b c d", 3), "a b\nc d"; got != want {
		t.Errorf("heading wrap = %q, want %q", got, want)
	}
	for md, want := range map[string]string{
		"####### x": "####### x", "#nospace": "#nospace", "## a ##": "a", "# a#": "a#",
	} {
		if got := plain(md, 40); got != want {
			t.Errorf("Render(%q) = %q, want %q", md, got, want)
		}
	}
}

func TestRenderInlineStyles(t *testing.T) {
	th := dark
	text := func() ansi.Style { return ansi.NewStyle().Foreground(th.Text) }
	tests := []struct {
		name, md string
		want     string
	}{
		{"plain uses Text", "hi", text().Render("hi")},
		{"italic", "*a*", text().Italic().Render("a")},
		{"bold", "**a**", text().Bold().Render("a")},
		{"bold italic", "***a***", text().Bold().Italic().Render("a")},
		{"code", "`c`", ansi.NewStyle().Foreground(th.Secondary).Render("c")},
		{"code keeps inner spaces", "`a b`", ansi.NewStyle().Foreground(th.Secondary).Render("a b")},
		{"link", "[t](u)", ansi.NewStyle().Foreground(th.Info).Underline().Render("t") + " " + ansi.NewStyle().Foreground(th.Muted).Render("(u)")},
		{"autolink shows url once", "<https://x.y>", ansi.NewStyle().Foreground(th.Info).Underline().Render("https://x.y")},
		{"link text equals url", "[u](u)", ansi.NewStyle().Foreground(th.Info).Underline().Render("u")},
		{"words are styled separately", "**a b**", text().Bold().Render("a") + " " + text().Bold().Render("b")},
		{"mixed", "a *b* c", text().Render("a") + " " + text().Italic().Render("b") + " " + text().Render("c")},
	}
	for _, tt := range tests {
		if got := Render(tt.md, 40, th); got != tt.want {
			t.Errorf("%s: Render(%q)\n got  %q\n want %q", tt.name, tt.md, got, tt.want)
		}
	}
}

func TestRenderLinks(t *testing.T) {
	tests := []struct{ md, want string }{
		{"[t](http://x)", "t (http://x)"},
		{`[t](u "title")`, "t (u)"},
		{"![alt](i.png)", "alt (i.png)"},
		{"see [a](u), ok", "see a (u), ok"},
		{"[a] [b]", "[a] [b]"},
		{"[a][b]", "[a][b]"},
		{"[^1]", "[^1]"},
		{"[a]()", "a"},
		{"[](u)", "u"},
	}
	for _, tt := range tests {
		if got := plain(tt.md, 60); got != tt.want {
			t.Errorf("Render(%q) = %q, want %q", tt.md, got, tt.want)
		}
	}
	// A long url is broken by column, not left overflowing.
	got := plain("[t](http://example.com/a/very/long/path)", 12)
	for _, l := range strings.Split(got, "\n") {
		if ansi.Width(l) > 12 {
			t.Errorf("line %q wider than 12", l)
		}
	}
}

func TestRenderLists(t *testing.T) {
	tests := []struct {
		name, md string
		width    int
		want     string
	}{
		{"bullets", "- a\n- b", 20, "• a\n• b"},
		{"star and plus", "* a\n\n+ b", 20, "• a\n\n• b"},
		{"ordered", "1. a\n2. b", 20, "1. a\n2. b"},
		{"ordered paren keeps delimiter", "1) a\n2) b", 20, "1) a\n2) b"},
		{"ordered starts where the list does", "5. a\n6. b", 20, "5. a\n6. b"},
		{"numbers right-aligned", "9. a\n10. b", 20, " 9. a\n10. b"},
		{"numbering ignores later numbers", "1. a\n7. b", 20, "1. a\n2. b"},
		{"nested", "- a\n  - b", 20, "• a\n  ◦ b"},
		{"nested ordered", "- a\n  1. b\n  2. c", 20, "• a\n  1. b\n  2. c"},
		{"third level is text", "- a\n  - b\n    - c", 20, "• a\n  ◦ b - c"},
		{"hanging indent when wrapped", "- aaa bbb ccc", 8, "• aaa\n  bbb\n  ccc"},
		{"ordered hanging indent", "10. aaa bbb", 9, "10. aaa\n    bbb"},
		{"empty item", "-", 20, "•"},
		{"empty item then item", "- \n- a", 20, "•\n• a"},
		{"kind change is a new list", "- a\n* b", 20, "• a\n\n• b"},
		{"two paragraphs in one item", "- a\n\n  b", 20, "• a\n\n  b"},
		{"list after paragraph", "text\n- a", 20, "text\n\n• a"},
		{"code in item", "1. step\n   ```\n   cmd\n   ```", 20,
			"1. step\n" + indent(codeBox("cmd", 17), "   ")},
		{"blank between items stays tight", "- a\n\n- b", 20, "• a\n• b"},
	}
	for _, tt := range tests {
		if got := plain(tt.md, tt.width); got != tt.want {
			t.Errorf("%s: Render(%q, %d)\n got  %q\n want %q", tt.name, tt.md, tt.width, got, tt.want)
		}
	}
}

func TestRenderListMarkerStyle(t *testing.T) {
	th := dark
	got := Render("- a", 20, th)
	want := ansi.NewStyle().Foreground(th.Primary).Render("• ") + ansi.NewStyle().Foreground(th.Text).Render("a")
	if got != want {
		t.Errorf("bullet = %q, want %q", got, want)
	}
	got = Render("1. a", 20, th)
	want = ansi.NewStyle().Foreground(th.Primary).Render("1. ") + ansi.NewStyle().Foreground(th.Text).Render("a")
	if got != want {
		t.Errorf("ordered = %q, want %q", got, want)
	}
}

func TestRenderQuotes(t *testing.T) {
	tests := []struct {
		name, md string
		width    int
		want     string
	}{
		{"quote", "> a", 20, "│ a"},
		{"quote lines wrap as one paragraph", "> a\n> b", 20, "│ a b"},
		{"quote paragraphs", "> a\n>\n> b", 20, "│ a\n│\n│ b"},
		{"nested quote", ">> a", 20, "│ │ a"},
		{"quote with list", "> - a\n> - b", 20, "│ • a\n│ • b"},
		{"quote with heading", "> # h\n> text", 20, "│ h\n│\n│ text"},
		{"empty quote", ">", 20, "│"},
		{"wraps at reduced width", "> aaa bbb", 7, "│ aaa\n│ bbb"},
		{"no lazy continuation", "> a\nb", 20, "│ a\n\nb"},
		{"quote with code", "> ```\n> c\n> ```", 20, "│ " + strings.ReplaceAll(codeBox("c", 18), "\n", "\n│ ")},
	}
	for _, tt := range tests {
		if got := plain(tt.md, tt.width); got != tt.want {
			t.Errorf("%s: Render(%q, %d)\n got  %q\n want %q", tt.name, tt.md, tt.width, got, tt.want)
		}
	}
	bar := ansi.NewStyle().Foreground(dark.Muted).Render("│")
	if got := Render("> a", 20, dark); !strings.HasPrefix(got, bar+" ") {
		t.Errorf("quote bar not Muted: %q", got)
	}
}

func TestRenderRules(t *testing.T) {
	for _, md := range []string{"---", "***", "___", "- - -", " * * * *", "-----"} {
		if got, want := plain(md, 6), "──────"; got != want {
			t.Errorf("Render(%q) = %q, want %q", md, got, want)
		}
	}
	want := ansi.NewStyle().Foreground(dark.Muted).Render("─────")
	if got := Render("---", 5, dark); got != want {
		t.Errorf("rule = %q, want Muted %q", got, want)
	}
	// Rules win over list markers.
	if got := plain("* * *", 4); got != "────" {
		t.Errorf("'* * *' = %q, want a rule", got)
	}
}

func TestRenderFencedCode(t *testing.T) {
	tests := []struct {
		name, md, code string
		width          int
	}{
		{"fence", "```\ncode\n```", "code", 14},
		{"info string ignored", "```go\ncode\n```", "code", 14},
		{"tilde fence", "~~~\ncode\n~~~", "code", 14},
		{"markdown inside is verbatim", "```\n# h\n**b**\n- x\n```", "# h\n**b**\n- x", 14},
		{"unclosed runs to end", "```\na\nb", "a\nb", 10},
		{"blank lines kept", "```\na\n\nb\n```", "a\n\nb", 10},
		{"long line truncated by CodeBlock", "```\n" + strings.Repeat("x", 50) + "\n```", strings.Repeat("x", 50), 12},
		{"empty fence", "```\n```", "", 8},
	}
	for _, tt := range tests {
		got := plain(tt.md, tt.width)
		if want := codeBox(tt.code, tt.width); got != want {
			t.Errorf("%s:\n got\n%s\n want\n%s", tt.name, got, want)
		}
	}
	// No line numbers, and code is not inline-parsed: styles come only
	// from CodeBlock (Text colour, no bold/italic/underline).
	out := Render("```\n**b** *i*\n```", 20, dark)
	for _, bad := range []string{"\x1b[1m", "\x1b[3m", "\x1b[4m", ";1;", ";3;", ";4;"} {
		if strings.Contains(out, bad) {
			t.Errorf("code content got inline styling %q: %q", bad, out)
		}
	}
	// A paragraph following the block is separated by one blank line.
	if got, want := plain("```\nc\n```\ntext", 10), codeBox("c", 10)+"\n\ntext"; got != want {
		t.Errorf("code then text:\n got %q\n want %q", got, want)
	}
}

func TestRenderOutOfScopeIsPlainText(t *testing.T) {
	tests := []struct{ name, md, want string }{
		{"html", "<b>x</b>", "<b>x</b>"},
		{"html block", "<div>\nhi\n</div>", "<div> hi </div>"},
		{"reference link", "[a][b]\n\n[b]: http://x", "[a][b]\n\n[b]: http://x"},
		{"footnote", "text[^1]\n\n[^1]: note", "text[^1]\n\n[^1]: note"},
		{"indented code is text", "    code here", "code here"},
		{"entity", "a &amp; b", "a &amp; b"},
		{"strikethrough", "~~x~~", "~~x~~"},
		{"task list", "- [ ] todo", "• [ ] todo"},
	}
	for _, tt := range tests {
		if got := plain(tt.md, 5+len(tt.want)); got != tt.want {
			t.Errorf("%s: Render(%q)\n got  %q\n want %q", tt.name, tt.md, got, tt.want)
		}
	}
}

func TestRenderSanitisesInput(t *testing.T) {
	tests := []struct{ name, md, want string }{
		{"csi colour", "a\x1b[31mred\x1b[0m", "ared"},
		{"osc title", "\x1b]0;evil\x07x", "x"},
		{"c1 csi", "a\u009b31mb", "a31mb"},
		{"nul and bell", "a\x00b\x07c", "abc"},
		{"lone escape before a control", "a\x1b\x01b", "ab"},
		{"esc plus a letter is a two-byte sequence", "a\x1bb", "a"},
		{"esc 7 is a two-byte sequence", "a\x1b7b", "ab"},
		{"tab is whitespace", "a\tb", "a b"},
		{"del", "a\x7fb", "ab"},
		{"invalid utf8 survives", "a\xffb", "a\uFFFDb"},
	}
	for _, tt := range tests {
		got := plain(tt.md, 40)
		if got != tt.want {
			t.Errorf("%s: Render(%q) = %q, want %q", tt.name, tt.md, got, tt.want)
		}
	}
	// Whatever the input, the only escapes in the output are our own SGR.
	out := Render("\x1b[2J\x1b[H\x1b]52;c;ZXZpbA==\x07# \x1b[31mhi\x1b[0m\n\n`\x1b[1m`", 40, dark)
	for i := 0; i < len(out); i++ {
		if out[i] == 0x1b {
			j := i + 1
			if j >= len(out) || out[j] != '[' {
				t.Fatalf("non-CSI escape at %d in %q", i, out)
			}
			for j++; j < len(out) && (out[j] == ';' || (out[j] >= '0' && out[j] <= '9')); j++ {
			}
			if j >= len(out) || out[j] != 'm' {
				t.Fatalf("non-SGR escape at %d in %q", i, out)
			}
		}
	}
	for _, r := range ansi.StripANSI(out) {
		if r < 0x20 && r != '\n' {
			t.Errorf("control character %U in output %q", r, out)
		}
	}
}

// --- properties ---

var corpus = []string{
	"# Title\n\nSome *emphasis*, **strong**, ***both***, `code` and [a link](http://example.com/very/long/path).",
	"- one\n- two\n  - nested item that is long enough to wrap around\n- three\n\n1. a\n2. b\n10. c",
	"> quoted text that wraps\n>\n> > nested quote\n> - list in quote",
	"```go\nfunc main() {\n\tfmt.Println(\"hello, 世界\")\n}\n```",
	"---\n\n***\n\ntext\n---",
	"你好世界你好世界你好世界 mixed with ascii words and 😀 emoji",
	"a  \nb\\\nc\n\n\n\nd",
	"1. step one\n   ```\n   cmd --flag\n   ```\n2. step two\n\n   second paragraph",
	strings.Repeat("word ", 60),
	strings.Repeat("x", 100),
	"| a | b |\n|---|---|\n| 1 | 2 |\n\n<div>html</div>\n\n[ref]: http://x",
	"*unclosed **emphasis and `code and [link](",
	"#\n##\n> \n-\n1.\n```\n",
}

// sgrOpen reports whether a line leaves SGR styling active at its end.
func sgrOpen(line string) bool {
	open := false
	for i := 0; i < len(line); i++ {
		if line[i] != 0x1b || i+1 >= len(line) || line[i+1] != '[' {
			continue
		}
		j := i + 2
		for j < len(line) && (line[j] == ';' || (line[j] >= '0' && line[j] <= '9')) {
			j++
		}
		if j < len(line) && line[j] == 'm' {
			params := line[i+2 : j]
			open = params != "" && params != "0"
		}
		i = j
	}
	return open
}

func checkInvariants(t *testing.T, md string, width int) {
	t.Helper()
	out := Render(md, width, dark)
	if out == "" {
		return
	}
	if strings.HasPrefix(out, "\n") || strings.HasSuffix(out, "\n") {
		t.Fatalf("Render(%q, %d) has a leading/trailing newline: %q", md, width, out)
	}
	if strings.Contains(out, "\n\n\n") {
		t.Fatalf("Render(%q, %d) has consecutive blank lines: %q", md, width, out)
	}
	for i, l := range strings.Split(out, "\n") {
		if w := ansi.Width(l); w > width {
			t.Fatalf("Render(%q, %d) line %d is %d wide: %q", md, width, i, w, l)
		}
		if sgrOpen(l) {
			t.Fatalf("Render(%q, %d) line %d leaves styling open: %q", md, width, i, l)
		}
	}
}

func TestRenderInvariantsOnCorpus(t *testing.T) {
	for _, md := range corpus {
		for w := 1; w <= 60; w++ {
			checkInvariants(t, md, w)
		}
		checkInvariants(t, md, 200)
	}
}

func TestRenderRandomInputNeverBreaksInvariants(t *testing.T) {
	alphabet := []string{"*", "_", "`", "[", "]", "(", ")", "<", ">", "#", "-", "+", ".", "1", "2", ">", "~", "!", " ", " ", "\n", "\n", "\\", "|", "a", "b", "word", "  ", "\t", "你", "😀", "http://x", "```", "---", "**", "1. ", "- ", "> ", "# "}
	rng := rand.New(rand.NewSource(1))
	for n := 0; n < 3000; n++ {
		var b strings.Builder
		for k := rng.Intn(40); k >= 0; k-- {
			b.WriteString(alphabet[rng.Intn(len(alphabet))])
		}
		md := b.String()
		for _, w := range []int{1, 2, 3, 5, 8, 20, 80} {
			checkInvariants(t, md, w)
		}
	}
}

func FuzzRender(f *testing.F) {
	for _, s := range corpus {
		f.Add(s)
	}
	f.Add("\x1b[31m*a*\x1b[0m")
	f.Add("- - - -\n* * *\n1. 2. 3.")
	f.Fuzz(func(t *testing.T, md string) {
		for _, w := range []int{1, 4, 17, 80} {
			checkInvariants(t, md, w)
		}
	})
}

func TestRenderUsesOnlyThemeColours(t *testing.T) {
	th := dark
	allowed := map[string]bool{}
	for _, c := range []ansi.Color{th.Text, th.Primary, th.Secondary, th.Info, th.Muted, th.BorderColor} {
		seq := ansi.NewStyle().Foreground(c).Render("x")
		allowed[strings.TrimSuffix(strings.TrimPrefix(seq, "\x1b["), "mx\x1b[0m")] = true
	}
	for _, md := range corpus {
		out := Render(md, 40, th)
		for i := 0; i < len(out); i++ {
			if out[i] != 0x1b {
				continue
			}
			j := strings.IndexByte(out[i:], 'm')
			params := out[i+2 : i+j]
			for _, p := range strings.Split(params, ";") {
				n, err := strconv.Atoi(p)
				if err != nil {
					continue
				}
				if (n >= 30 && n <= 37) || (n >= 90 && n <= 97) {
					if !allowed[p] {
						t.Errorf("colour code %s not from the theme in Render(%q)", p, md)
					}
				}
			}
			i += j
		}
	}
}

// growthRatio renders gen(n) and gen(4n) and returns how many times longer
// the larger one took. Linear rendering gives about 4; quadratic about 16.
// Each size is timed three times and the fastest run is used, so a noisy
// machine (GC, a busy CI runner) inflates both sides alike instead of
// failing one absolute wall-clock limit.
func growthRatio(gen func(n int) string, n int, render func(string)) float64 {
	best := func(md string) time.Duration {
		min := time.Duration(1<<63 - 1)
		for i := 0; i < 3; i++ {
			start := time.Now()
			render(md)
			if d := time.Since(start); d < min {
				min = d
			}
		}
		return min
	}
	small, large := best(gen(n)), best(gen(4*n))
	if small < 200*time.Microsecond {
		small = 200 * time.Microsecond // below timer noise; don't divide by ~0
	}
	return float64(large) / float64(small)
}

// growthAttempts is how many times a growth measurement may be repeated.
// Noise (a busy runner, coarse timers) only ever distorts a ratio for a
// while, whereas real super-linear rendering gives a high ratio every time,
// so the lowest ratio over a few attempts still exposes it while a one-off
// noisy sample no longer fails the run.
const growthAttempts = 5

// minGrowthRatio is the lowest growthRatio over growthAttempts attempts.
func minGrowthRatio(gen func(n int) string, n int, render func(string)) float64 {
	lowest := growthRatio(gen, n, render)
	for i := 1; i < growthAttempts && lowest > maxGrowthRatio; i++ {
		if r := growthRatio(gen, n, render); r < lowest {
			lowest = r
		}
	}
	return lowest
}

// maxGrowthRatio is the failure threshold: comfortably above the ~4x a
// linear renderer shows (allowing for cache effects and noise) and far
// below the ~16x of a quadratic one.
const maxGrowthRatio = 10

// spin is a stand-in renderer whose cost is work(len(s)).
func spin(work func(n int) int) func(string) {
	return func(s string) {
		x := 0
		for i, n := 0, work(len(s)); i < n; i++ {
			x += i
		}
		sink = x
	}
}

var sink int

func TestGrowthRatioSeparatesLinearFromQuadratic(t *testing.T) {
	gen := func(n int) string { return strings.Repeat("x", n) }
	// Noise can push a linear ratio up or a quadratic one down, so take the
	// lowest linear and the highest quadratic ratio over a few attempts. The
	// harness only has to separate the two around maxGrowthRatio.
	linearWork := spin(func(n int) int { return n * 40 })
	quadraticWork := spin(func(n int) int { return n * n })
	linear := growthRatio(gen, 500000, linearWork)
	quadratic := growthRatio(gen, 3000, quadraticWork)
	for i := 1; i < growthAttempts && (linear >= maxGrowthRatio || quadratic <= maxGrowthRatio); i++ {
		if r := growthRatio(gen, 500000, linearWork); r < linear {
			linear = r
		}
		if r := growthRatio(gen, 3000, quadraticWork); r > quadratic {
			quadratic = r
		}
	}
	if linear >= maxGrowthRatio {
		t.Errorf("linear renderer ratio = %.1f, want well under the %d failure threshold", linear, maxGrowthRatio)
	}
	if quadratic <= maxGrowthRatio {
		t.Errorf("quadratic renderer ratio = %.1f would not exceed the %d failure threshold", quadratic, maxGrowthRatio)
	}
}

// Hostile input must not make Render super-linear. Each case would blow up
// with the original quadratic emphasis matching, unbounded link scans, or
// unlimited quote nesting. Cases are judged by growth, not by wall-clock:
// see growthRatio.
func TestRenderPathologicalInputIsFast(t *testing.T) {
	rep := strings.Repeat
	cases := map[string]struct {
		n   int
		gen func(n int) string
	}{
		"emphasis run":       {10000, func(n int) string { return rep("*a", n) }},
		"underscore pairs":   {10000, func(n int) string { return rep("_a_ ", n) }},
		"mixed delimiters":   {8000, func(n int) string { return rep("*_a", n) }},
		"emphasis and links": {5000, func(n int) string { return rep("*[a](u)", n) }},
		"open brackets":      {15000, func(n int) string { return rep("[", n) }},
		"unclosed links":     {8000, func(n int) string { return rep("[a](", n) }},
		"unclosed titles":    {5000, func(n int) string { return rep(`[a](u "`, n) }},
		"nested link text":   {5000, func(n int) string { return rep("[", n) + "a" + rep("](u)", n) }},
		"backticks":          {10000, func(n int) string { return rep("`a", n) }},
		"deep quotes":        {15000, func(n int) string { return rep(">", n) + "a" }},
		"deep lists":         {8000, func(n int) string { return rep("- ", n) + "a" }},
		"many list items":    {8000, func(n int) string { return rep("- a\n", n) }},
		"many fences":        {8000, func(n int) string { return rep("```\n", n) }},
		"autolink openers":   {8000, func(n int) string { return rep("<a:", n) }},
		"one huge word":      {125000, func(n int) string { return rep("x", n) }},
		"one huge paragraph": {25000, func(n int) string { return rep("word ", n) }},
	}
	for name, c := range cases {
		ratio := minGrowthRatio(c.gen, c.n, func(md string) { renderUncached(md, 80, dark, Options{}) })
		t.Logf("%-20s 4x input -> %.1fx time", name, ratio)
		if ratio > maxGrowthRatio {
			t.Errorf("%s: 4x input took %.1fx longer, want about 4x (limit %dx): super-linear", name, ratio, maxGrowthRatio)
		}
	}
}
