package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

const (
	goSample   = "package main\n\n// Greet says hi.\nfunc Greet(n string) (int, error) {\n\tif n == \"\" {\n\t\treturn 0, nil\n\t}\n\tconst raw = `a\nb`\n\treturn 0x2A + 1, nil /* done */\n}\n"
	jsonSample = "{\n  \"name\": \"tui\",\n  \"n\": 3.5,\n  \"ok\": true,\n  \"tags\": [\"a\", null]\n}\n"
	shSample   = "#!/bin/sh\n# build\nif [ -n \"$HOME\" ]; then\n  echo ${X:-1} $1\nfi\n"
	wideSample = "// 你好世界 wide 日本語 text that is long enough to be cut\nvar s = \"é日本語\"\n"
)

var langSamples = map[string]string{"go": goSample, "json": jsonSample, "sh": shSample}

// When highlighting does not apply (no or unknown language), the output is
// byte-identical to CodeBlock, at every width and gutter setting.
func TestCodeBlockLangUnsupportedIsByteIdentical(t *testing.T) {
	for _, lang := range []string{"", "cobol", "text", "Go2"} {
		for _, code := range []string{goSample, "", "x", "a\tb\r\nc\n"} {
			for w := -1; w <= 40; w++ {
				for _, nums := range []bool{false, true} {
					if got, want := CodeBlockLang(code, lang, w, nums, theme.DarkTheme()), CodeBlock(code, w, nums, theme.DarkTheme()); got != want {
						t.Fatalf("lang %q width %d nums %v differs from CodeBlock:\n%q\n%q", lang, w, nums, got, want)
					}
				}
			}
		}
	}
}

// Highlighting only adds colour: with the styling removed, the layout is the
// same as the plain CodeBlock, including gutter, padding and ellipsis.
func TestCodeBlockLangKeepsCodeBlocksLayout(t *testing.T) {
	samples := map[string]string{"go": goSample + wideSample, "json": jsonSample, "sh": shSample}
	for lang, code := range samples {
		for w := -1; w <= 70; w++ {
			for _, nums := range []bool{false, true} {
				got := ansi.StripANSI(CodeBlockLang(code, lang, w, nums, theme.DarkTheme()))
				want := ansi.StripANSI(CodeBlock(code, w, nums, theme.DarkTheme()))
				if got != want {
					t.Fatalf("%s width %d nums %v layout differs:\n%s\n--- want\n%s", lang, w, nums, got, want)
				}
			}
		}
	}
}

// The highlighted block is exactly the allotted size at any width: width
// columns per row, one row per source line plus two border rows.
func TestCodeBlockLangReturnsExactlyTheAllottedSize(t *testing.T) {
	for lang, code := range langSamples {
		code += wideSample
		lines := len(strings.Split(strings.TrimSuffix(code, "\n"), "\n"))
		for w := 1; w <= 80; w++ {
			for _, nums := range []bool{false, true} {
				out := CodeBlockLang(code, lang, w, nums, theme.DarkTheme())
				rows := strings.Split(out, "\n")
				wantRows := lines
				if w >= codeBlockMinWidth {
					wantRows += 2
				}
				if len(rows) != wantRows {
					t.Fatalf("%s width %d: %d rows, want %d", lang, w, len(rows), wantRows)
				}
				for i, r := range rows {
					if got := ansi.Width(r); got != w && !(w < codeBlockMinWidth && got <= w) {
						t.Fatalf("%s width %d nums %v row %d is %d wide: %q", lang, w, nums, i, got, ansi.StripANSI(r))
					}
				}
			}
		}
		if got := CodeBlockLang(code, lang, 0, true, theme.DarkTheme()); got != "" {
			t.Errorf("%s width 0 = %q, want empty", lang, got)
		}
	}
}

// Every class is drawn in its own theme colour.
func TestCodeBlockLangColoursTokensByClass(t *testing.T) {
	th := theme.DarkTheme()
	paint := func(c ansi.Color, s string) string { return ansi.NewStyle().Foreground(c).Render(s) }
	for _, tc := range []struct {
		lang, code, token string
		color             ansi.Color
	}{
		{"go", "func f() {}", "func", th.Primary},
		{"go", "var x int", "int", th.Info},
		{"go", `s := "hi"`, `"hi"`, th.Success},
		{"go", "n := 42", "42", th.Warning},
		{"go", "// note", "// note", th.Muted},
		{"go", "x = nil", "nil", th.Secondary},
		{"json", `{"k": 1}`, `"k"`, th.Info},
		{"json", `{"k": "v"}`, `"v"`, th.Success},
		{"json", `[true]`, "true", th.Secondary},
		{"sh", "if x; then", "then", th.Primary},
		{"sh", "echo $HOME", "$HOME", th.Info},
		{"sh", "# c", "# c", th.Muted},
	} {
		out := CodeBlockLang(tc.code, tc.lang, 40, false, th)
		if want := paint(tc.color, tc.token); !strings.Contains(out, want) {
			t.Errorf("%s %q: token %q not painted %q in %q", tc.lang, tc.code, tc.token, want, out)
		}
	}
}

// Multi-line tokens keep their colour on every line they span.
func TestCodeBlockLangMultiLineTokens(t *testing.T) {
	th := theme.DarkTheme()
	out := CodeBlockLang("/* a\nb */\nx := `p\nq`", "go", 30, false, th)
	for _, tok := range []string{"/* a", "b */"} {
		if !strings.Contains(out, ansi.NewStyle().Foreground(th.Muted).Render(tok)) {
			t.Errorf("comment part %q not Muted", tok)
		}
	}
	for _, tok := range []string{"`p", "q`"} {
		if !strings.Contains(out, ansi.NewStyle().Foreground(th.Success).Render(tok)) {
			t.Errorf("raw string part %q not painted as a string", tok)
		}
	}
}

func TestCodeBlockLangLanguageNamesAreCaseInsensitive(t *testing.T) {
	a := CodeBlockLang("func f() {}", "Go", 30, false, theme.DarkTheme())
	b := CodeBlockLang("func f() {}", "golang", 30, false, theme.DarkTheme())
	c := CodeBlockLang("func f() {}", "go", 30, false, theme.DarkTheme())
	if a != c || b != c {
		t.Error("language aliases give different output")
	}
}
