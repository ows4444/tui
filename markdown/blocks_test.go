package markdown

import (
	"fmt"
	"strings"
	"testing"
)

// dump renders the block tree as a compact string for comparison:
// P(text) H<n>(text) CODE(text) HR Q(...) UL[...] OL<start>[...]
func dump(bs []block) string {
	var parts []string
	for _, b := range bs {
		switch b := b.(type) {
		case paragraph:
			parts = append(parts, "P("+b.text+")")
		case heading:
			parts = append(parts, fmt.Sprintf("H%d(%s)", b.level, b.text))
		case codeBlock:
			parts = append(parts, "CODE("+b.text+")")
		case rule:
			parts = append(parts, "HR")
		case quote:
			parts = append(parts, "Q("+dump(b.children)+")")
		case list:
			var items []string
			for _, it := range b.items {
				items = append(items, "<"+dump(it)+">")
			}
			name := "UL"
			if b.ordered {
				name = fmt.Sprintf("OL%d%c", b.start, b.delim)
			}
			parts = append(parts, name+"["+strings.Join(items, "")+"]")
		default:
			parts = append(parts, fmt.Sprintf("?%T", b))
		}
	}
	return strings.Join(parts, " ")
}

func TestParseBlocks(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"empty", "", ""},
		{"blank lines", "\n  \n\n", ""},

		// Paragraphs.
		{"paragraph", "a", "P(a)"},
		{"lines join", "a\nb", "P(a\nb)"},
		{"blank separates", "a\n\nb", "P(a) P(b)"},
		{"leading spaces trimmed", "  a\n   b", "P(a\nb)"},
		{"trailing spaces kept for hard break", "a  \nb", "P(a  \nb)"},
		{"final line trailing spaces trimmed", "a  ", "P(a)"},
		{"indented four is text not code", "    code", "P(code)"},

		// Headings.
		{"h1", "# a", "H1(a)"},
		{"h6", "###### f", "H6(f)"},
		{"seven hashes is text", "####### x", "P(####### x)"},
		{"no space is text", "#nospace", "P(#nospace)"},
		{"closing hashes", "## a ##", "H2(a)"},
		{"closing hashes need a space", "# a#", "H1(a#)"},
		{"empty heading", "#", "H1()"},
		{"all hashes closing", "# #", "H1()"},
		{"three spaces indent ok", "   # a", "H1(a)"},
		{"four spaces indent is text", "    # a", "P(# a)"},
		{"heading interrupts paragraph", "a\n# b\nc", "P(a) H1(b) P(c)"},

		// Rules.
		{"rule dashes", "---", "HR"},
		{"rule stars", "***", "HR"},
		{"rule underscores", "___", "HR"},
		{"rule with spaces", "- - -", "HR"},
		{"rule star spaces", " * * * *", "HR"},
		{"two dashes is text", "--", "P(--)"},
		{"mixed chars is text", "-*-", "P(-*-)"},
		{"dashes under text are a setext h2", "text\n---", "H2(text)"},
		{"equals under text are a setext h1", "text\n===", "H1(text)"},

		// Fences.
		{"fence", "```\na\n```", "CODE(a)"},
		{"fence info ignored", "```go\nx\n```", "CODE(x)"},
		{"tilde fence", "~~~\na\n~~~", "CODE(a)"},
		{"unclosed fence runs to end", "```\na\nb", "CODE(a\nb)"},
		{"longer closing ok", "```\na\n`````", "CODE(a)"},
		{"shorter closing does not close", "````\n```\n````", "CODE(```)"},
		{"tilde does not close backtick", "```\n~~~\n```", "CODE(~~~)"},
		{"blank lines kept", "```\na\n\nb\n```", "CODE(a\n\nb)"},
		{"markdown inside is verbatim", "```\n# not a heading\n- not a list\n```", "CODE(# not a heading\n- not a list)"},
		{"indented fence strips indent", " ```\n  a\n ```", "CODE( a)"},
		{"backtick in info is not a fence", "```a`b", "P(```a`b)"},
		{"empty fence", "```\n```", "CODE()"},
		{"fence interrupts paragraph", "a\n```\nc\n```\nb", "P(a) CODE(c) P(b)"},
		{"closing fence may have trailing spaces", "```\na\n```  ", "CODE(a)"},

		// Block quotes.
		{"quote", "> a", "Q(P(a))"},
		{"quote no space", ">a", "Q(P(a))"},
		{"quote lines", "> a\n> b", "Q(P(a\nb))"},
		{"no lazy continuation", "> a\nb", "Q(P(a)) P(b)"},
		{"nested quote", ">> a", "Q(Q(P(a)))"},
		{"quote with blocks", "> # h\n> - x", "Q(H1(h) UL[<P(x)>])"},
		{"quote paragraphs", "> a\n>\n> b", "Q(P(a) P(b))"},
		{"quote with fence", "> ```\n> c\n> ```", "Q(CODE(c))"},
		{"quote interrupts paragraph", "a\n> b", "P(a) Q(P(b))"},

		// Lists.
		{"bullets", "- a\n- b", "UL[<P(a)><P(b)>]"},
		{"star bullets", "* a\n* b", "UL[<P(a)><P(b)>]"},
		{"plus bullets", "+ a", "UL[<P(a)>]"},
		{"ordered", "1. a\n2. b", "OL1.[<P(a)><P(b)>]"},
		{"ordered paren", "3) x\n4) y", "OL3)[<P(x)><P(y)>]"},
		{"ordered start kept", "5. a\n6. b", "OL5.[<P(a)><P(b)>]"},
		{"bullet char change starts new list", "- a\n* b", "UL[<P(a)>] UL[<P(b)>]"},
		{"ordered then bullet", "1. a\n- b", "OL1.[<P(a)>] UL[<P(b)>]"},
		{"delimiter change starts new list", "1. a\n2) b", "OL1.[<P(a)>] OL2)[<P(b)>]"},
		{"nested", "- a\n  - b", "UL[<P(a) UL[<P(b)>]>]"},
		{"nested four-space", "- a\n    - b", "UL[<P(a) UL[<P(b)>]>]"},
		{"nested ordered in bullet", "- a\n  1. b", "UL[<P(a) OL1.[<P(b)>]>]"},
		{"third level is text", "- a\n  - b\n    - c", "UL[<P(a) UL[<P(b\n- c)>]>]"},
		{"blank between items", "- a\n\n- b", "UL[<P(a)><P(b)>]"},
		{"list then paragraph", "- a\n\nb", "UL[<P(a)>] P(b)"},
		{"continuation line", "- a\n  b", "UL[<P(a\nb)>]"},
		{"lazy continuation", "- a\nb", "UL[<P(a\nb)>]"},
		{"two paragraphs in item", "- a\n\n  b", "UL[<P(a) P(b)>]"},
		{"empty item", "-", "UL[<>]"},
		{"empty item with space", "- \n- a", "UL[<><P(a)>]"},
		{"code in item", "1. step\n   ```\n   cmd\n   ```", "OL1.[<P(step) CODE(cmd)>]"},
		{"rule ends list", "- a\n---", "UL[<P(a)>] HR"},
		{"marker needs space", "-a", "P(-a)"},
		{"ordered marker needs space", "1.a", "P(1.a)"},
		{"ten digits is not a marker", "1234567890. a", "P(1234567890. a)"},
		{"wide marker content offset", "10. a\n    b", "OL10.[<P(a\nb)>]"},
		{"bullet interrupts paragraph", "para\n- a", "P(para) UL[<P(a)>]"},
		{"ordered one interrupts", "para\n1. a", "P(para) OL1.[<P(a)>]"},
		{"ordered other does not interrupt", "para\n2. a", "P(para\n2. a)"},
		{"rule beats list marker", "* * *", "HR"},
		{"list in quote", "> - a\n> - b", "Q(UL[<P(a)><P(b)>])"},
		{"quote in list", "- > a", "UL[<Q(P(a))>]"},
		{"heading in item", "- # h", "UL[<H1(h)>]"},
		{"sibling indented one space", "- a\n - b", "UL[<P(a)><P(b)>]"},
		{"five spaces after marker is code-ish offset", "-     a", "UL[<P(a)>]"},

		// Mixed.
		{"document", "# t\n\ntext\n\n```\nc\n```\n\n- a", "H1(t) P(text) CODE(c) UL[<P(a)>]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dump(parseBlocks(strings.Split(tt.in, "\n"), 0, 0)); got != tt.want {
				t.Errorf("parseBlocks(%q)\n got  %s\n want %s", tt.in, got, tt.want)
			}
		})
	}
}
