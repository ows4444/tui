package highlight

import "testing"

// Criterion #113: script and style content goes to the JS and CSS lexers,
// whatever the tag's case, and a self-closing or src-only script does not
// swallow the rest of the page.
func TestHTMLEmbedsScriptAndStyle(t *testing.T) {
	lines := Lines("html", "<SCRIPT type=module> let n = 1 </SCRIPT>\n<style>p { color: red }</style>")
	if got := classOf(t, lines, "let"); got != Keyword {
		t.Errorf("script content: let is %d, want Keyword", got)
	}
	if got := classOf(t, lines, "color"); got != Key {
		t.Errorf("style content: color is %d, want Key", got)
	}
	if got := classOf(t, lines, "</SCRIPT>"); got != Keyword {
		t.Errorf("closing tag is %d, want Keyword", got)
	}
	lines = Lines("html", `<script src="a.js"/><p>let</p>`)
	if got := classOf(t, lines, "let"); got != Plain {
		t.Errorf("after a self-closing script, text is %d, want Plain", got)
	}
	// XML has no script semantics: the element's text stays plain.
	if got := classOf(t, Lines("xml", "<script>let</script>"), "let"); got != Plain {
		t.Errorf("xml script content is %d, want Plain", got)
	}
}

// Language-specific tokens the goldens rely on, checked one at a time.
func TestNewLanguageTokens(t *testing.T) {
	cases := []struct {
		lang, code, text string
		want             Class
	}{
		{"c#", `var p = @"C:\x""y";`, `@"C:\x""y"`, String},
		{"c#", `Task.Run(Work)`, ".Run(", Plain}, // a call, not a type
		{"kotlin", "/* a /* b */ c */ val", "/* a /* b */ c */", Comment},
		{"swift", "#if os(macOS)", "#if", Keyword},
		{"php", "<?php $x = NULL;", "$x", Variable},
		{"php", "<?php $x = NULL;", "NULL", Literal},
		{"php", "#[Route('/')]", "#[", Keyword},
		{"ruby", "attr_reader :name", ":name", Literal},
		{"ruby", "Foo::Bar", "Bar", Type},
		{"ruby", "h = { key: 1 }", "key:", Key},
		{"ruby", "x.empty?", "x.empty?", Plain}, // one plain span: the ? belongs to the name
		{"lua", "--[==[ c ]==] x", "--[==[ c ]==]", Comment},
		{"lua", "s = [[long]]", "[[long]]", String},
		{"css", "@media (min-width: 1px) { a:hover { b: 0 } }", "min-width", Key},
		{"css", "@media (min-width: 1px) { a:hover { b: 0 } }", ":hover", Keyword},
		{"css", "a { background: url(x.png) }", ": url(x.png) }", Plain}, // a function name is not a selector type
		{"ini", "[core]\nbare = yes", "yes", Literal},
		{"makefile", "\techo $${HOME} # c", "$${HOME}", Variable},
		{"makefile", "\techo ok # c's", "# c's", Comment},
		{"makefile", "a: b ## doc", "## doc", Comment},
	}
	for _, c := range cases {
		if got := classOf(t, Lines(c.lang, c.code), c.text); got != c.want {
			t.Errorf("%s %q: %q is %d, want %d", c.lang, c.code, c.text, got, c.want)
		}
	}
}
