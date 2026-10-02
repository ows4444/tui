package markdown

import (
	"strings"
	"testing"
)

// dbg renders pieces compactly: plain text as-is, styled runs as
// {flags:text} (b bold, i italic, c code, l link, m muted), a hard break
// as ⏎. Adjacent pieces with equal attrs are merged so expectations don't
// depend on how the parser happened to split text.
func dbg(ps []piece) string {
	var out strings.Builder
	var cur attrs
	var buf strings.Builder
	have := false
	flush := func() {
		if !have {
			return
		}
		flags := ""
		if cur.bold {
			flags += "b"
		}
		if cur.italic {
			flags += "i"
		}
		if cur.code {
			flags += "c"
		}
		if cur.link {
			flags += "l"
		}
		if cur.muted {
			flags += "m"
		}
		if flags == "" {
			out.WriteString(buf.String())
		} else {
			out.WriteString("{" + flags + ":" + buf.String() + "}")
		}
		buf.Reset()
		have = false
	}
	for _, p := range ps {
		if p.brk {
			flush()
			out.WriteString("⏎")
			continue
		}
		if have && p.a != cur {
			flush()
		}
		cur, have = p.a, true
		buf.WriteString(p.text)
	}
	flush()
	return out.String()
}

func TestParseInline(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"plain", "hello world", "hello world"},
		{"empty", "", ""},

		// Emphasis.
		{"em star", "*a*", "{i:a}"},
		{"em underscore", "_a_", "{i:a}"},
		{"strong star", "**a**", "{b:a}"},
		{"strong underscore", "__a__", "{b:a}"},
		{"strong+em", "***a***", "{bi:a}"},
		{"em in text", "a *b* c", "a {i:b} c"},
		{"strong containing em", "**a *b* c**", "{b:a }{bi:b}{b: c}"},
		{"em containing strong", "*a **b** c*", "{i:a }{bi:b}{i: c}"},
		{"rule of three", "*foo**bar**baz*", "{i:foo}{bi:bar}{i:baz}"},
		{"star intraword", "foo*bar*baz", "foo{i:bar}baz"},
		{"underscore intraword is literal", "snake_case_name", "snake_case_name"},
		{"underscore intraword strong is literal", "a__b__c", "a__b__c"},
		{"underscore intraword cannot open", "foo_bar_ baz", "foo_bar_ baz"},
		{"underscore intraword cannot close", "_foo bar_baz", "_foo bar_baz"},
		{"underscore emphasis containing underscore", "_foo_bar_", "{i:foo_bar}"},
		{"digits with underscores", "5_000_000 _x_", "5_000_000 {i:x}"},
		{"underscore at word edges", "_a_ and _b_", "{i:a} and {i:b}"},
		{"underscore next to punctuation", "(_a_)", "({i:a})"},
		{"unmatched opener", "*a", "*a"},
		{"spaced stars are literal", "a * b *", "a * b *"},
		{"extra opener stays literal", "**a*", "*{i:a}"},
		{"extra closer stays literal", "*a**", "{i:a}*"},
		{"emphasis needs non-space after opener", "* a*", "* a*"},
		{"punctuation flanking", "(*a*)", "({i:a})"},
		{"multibyte", "*héllo 你好*", "{i:héllo 你好}"},

		// Escapes.
		{"escaped stars", `\*not\*`, "*not*"},
		{"escaped backslash", `a\\b`, `a\b`},
		{"backslash before letter stays", `\a`, `\a`},
		{"escaped bracket", `\[a](b)`, "[a](b)"},
		{"escaped backtick", "\\`a\\`", "`a`"},

		// Code spans.
		{"code", "`a *b*`", "{c:a *b*}"},
		{"code double tick", "``a`b``", "{c:a`b}"},
		{"code strips one padding space", "` a `", "{c:a}"},
		{"code keeps all-space content", "`  `", "{c:  }"},
		{"code newline becomes space", "`a\nb`", "{c:a b}"},
		{"unmatched backtick literal", "`a", "`a"},
		{"mismatched run lengths", "``a`", "``a`"},
		{"code beside text", "x `c` y", "x {c:c} y"},
		{"emphasis around code", "*`c`*", "{ic:c}"},
		{"stars inside code are literal", "`**`", "{c:**}"},

		// Breaks.
		{"soft break", "a\nb", "a b"},
		{"hard break spaces", "a  \nb", "a⏎b"},
		{"hard break backslash", "a\\\nb", "a⏎b"},
		{"single trailing space is soft", "a \nb", "a b"},

		// Links.
		{"link", "[t](http://x)", "{l:t} {m:(http://x)}"},
		{"link with title", `[t](u "the title")`, "{l:t} {m:(u)}"},
		{"link single-quoted title", "[t](u 'x')", "{l:t} {m:(u)}"},
		{"link paren title", "[t](u (x))", "{l:t} {m:(u)}"},
		{"link text equals url", "[http://x](http://x)", "{l:http://x}"},
		{"autolink", "<https://a.b/c>", "{l:https://a.b/c}"},
		{"image", "![alt](u.png)", "{l:alt} {m:(u.png)}"},
		{"link with emphasis", "[*e*](u)", "{il:e} {m:(u)}"},
		{"emphasis around link", "*[a](u)*", "{il:a}{i: }{im:(u)}"},
		{"empty destination", "[a]()", "{l:a}"},
		{"empty text", "[](u)", "{l:u}"},
		{"angle destination", "[a](<u v>)", "{l:a} {m:(u v)}"},
		{"nested brackets in text", "[a [b] c](u)", "{l:a [b] c} {m:(u)}"},
		{"escaped bracket in text", `[a\]b](u)`, "{l:a]b} {m:(u)}"},
		{"parens in url", "[a](u(1))", "{l:a} {m:(u(1))}"},
		{"escaped paren in url", `[a](u\)x)`, "{l:a} {m:(u)x)}"},
		{"link before text", "see [t](u). ok", "see {l:t} {m:(u)}. ok"},
		{"unmatched bracket", "[a] b", "[a] b"},
		{"missing paren", "[a](b", "[a](b"},
		{"no destination", "[a] (b)", "[a] (b)"},
		{"lone bang", "hi! [a](u)", "hi! {l:a} {m:(u)}"},
		{"backticks hide brackets", "[a`]`b](u)", "{l:a}{cl:]}{l:b} {m:(u)}"},
		{"not an autolink", "a < b > c", "a < b > c"},
		{"autolink needs scheme", "<a>", "<a>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dbg(parseInline(tt.in)); got != tt.want {
				t.Errorf("parseInline(%q)\n got  %s\n want %s", tt.in, got, tt.want)
			}
		})
	}
}
