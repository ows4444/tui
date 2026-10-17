package markdown

import "testing"

func TestTaskLists(t *testing.T) {
	for _, tt := range []struct{ name, md, want string }{
		{"open and done", "- [ ] todo\n- [x] done\n- [X] also", "[ ] todo\n[x] done\n[x] also"},
		{"mixed with ordinary", "- [ ] a\n- plain", "[ ] a\n• plain"},
		{"no space after the box is text", "- [x]nope", "• [x]nope"},
		{"empty task", "- [ ]", "[ ]"},
		{"ordered lists have none", "1. [ ] a", "1. [ ] a"},
		{"nested", "- [ ] a\n  - [x] b", "[ ] a\n    [x] b"},
		{"wraps under the box", "- [ ] aa bb", "[ ] aa\n    bb"},
	} {
		width := 20
		if tt.name == "wraps under the box" {
			width = 8
		}
		if got := plain(tt.md, width); got != tt.want {
			t.Errorf("%s: Render(%q)\n got  %q\n want %q", tt.name, tt.md, got, tt.want)
		}
	}
}

func TestReferenceLinks(t *testing.T) {
	for _, tt := range []struct{ name, md, want string }{
		{"full", "see [the docs][d]\n\n[d]: http://x.test/docs", "see the docs (http://x.test/docs)"},
		{"collapsed", "[docs][]\n\n[docs]: http://x.test", "docs (http://x.test)"},
		{"shortcut", "[docs]\n\n[docs]: http://x.test", "docs (http://x.test)"},
		{"label case and space", "[a][  My   LABEL ]\n\n[my label]: http://y", "a (http://y)"},
		{"title is dropped", "[a][b]\n\n[b]: http://x \"A title\"", "a (http://x)"},
		{"angle brackets", "[a][b]\n\n[b]: <http://x/a b>", "a (http://x/a b)"},
		{"definition may come first", "[b]: http://x\n\n[a][b]", "a (http://x)"},
		{"undefined stays text", "[a][nope]", "[a][nope]"},
		{"first definition wins", "[a][b]\n\n[b]: http://one\n[b]: http://two", "a (http://one)"},
		{"image", "![pic][p]\n\n[p]: http://x/p.png", "pic (http://x/p.png)"},
		{"inline link is untouched", "[a](http://z) and [a][b]\n\n[b]: http://x", "a (http://z) and a (http://x)"},
		{"code span is untouched", "`[a][b]`\n\n[b]: http://x", "[a][b]"},
		{"escaped bracket is text, the next reference still links", "\\[a][b]\n\n[b]: http://x", "[a]b (http://x)"},
		{"fenced code keeps its definition", "```\n[a][b]\n[b]: http://x\n```", "" /* checked below */},
		{"footnote is not a link", "text[^1]\n\n[^1]: note", "text[^1]\n\n[^1]: note"},
		{"definition with junk after is text", "[b]: http://x extra", "[b]: http://x extra"},
	} {
		if tt.name == "fenced code keeps its definition" {
			out := plain(tt.md, 40)
			if !contains(out, "[a][b]") || !contains(out, "[b]: http://x") {
				t.Errorf("%s: fenced code was rewritten: %q", tt.name, out)
			}
			continue
		}
		if got := plain(tt.md, 60); got != tt.want {
			t.Errorf("%s: Render(%q)\n got  %q\n want %q", tt.name, tt.md, got, tt.want)
		}
	}
}

func TestEntities(t *testing.T) {
	for _, tt := range []struct{ name, md, want string }{
		{"named", "a &amp; b &lt;i&gt; &copy;", "a & b <i> ©"},
		{"decimal and hex", "&#8212; &#x2014;", "— —"},
		{"unknown stays", "&nope; and AT&T", "&nope; and AT&T"},
		{"no semicolon stays", "&amp b", "&amp b"},
		{"control stays literal", "&#27;[31m", "&#27;[31m"},
		{"nul stays literal", "&#0;", "&#0;"},
		{"code is not decoded", "`&amp;`", "&amp;"},
		{"decoded text is not markup", "&#42;a&#42;", "*a*"},
	} {
		if got := plain(tt.md, 40); got != tt.want {
			t.Errorf("%s: Render(%q)\n got  %q\n want %q", tt.name, tt.md, got, tt.want)
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
