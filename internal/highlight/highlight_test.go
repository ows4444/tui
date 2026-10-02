package highlight

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// classOf returns the class of the first span whose text is exactly text.
func classOf(t *testing.T, lines [][]Span, text string) Class {
	t.Helper()
	for _, l := range lines {
		for _, sp := range l {
			if sp.Text == text {
				return sp.Class
			}
		}
	}
	t.Fatalf("no span with text %q in %v", text, lines)
	return Plain
}

func TestSupported(t *testing.T) {
	for _, l := range []string{"go", "Go", " golang ", "json", "sh", "bash", "shell", "zsh", "yaml", "YML", "python", "py"} {
		if !Supported(l) {
			t.Errorf("Supported(%q) = false", l)
		}
	}
	for _, l := range []string{"", "cobol", "text"} {
		if Supported(l) {
			t.Errorf("Supported(%q) = true", l)
		}
	}
}

func TestGoTokens(t *testing.T) {
	const src = "package main\n\n// note\nfunc f(x int) string {\n\treturn \"hi\" + `raw\nline` // end\n}\n/* a\nb */ var n = 0x1F + 3.5e-2 + iota + 'c'\n"
	lines := Lines("go", src)
	for text, want := range map[string]Class{
		"package": Keyword, "func": Keyword, "return": Keyword, "var": Keyword,
		"int": Type, "string": Type,
		"// note": Comment, "// end": Comment,
		`"hi"`: String, "'c'": String,
		"0x1F": Number, "3.5e-2": Number,
		"iota": Literal,
	} {
		if got := classOf(t, lines, text); got != want {
			t.Errorf("%q class = %d, want %d", text, got, want)
		}
	}
	// Neighbouring plain text merges into one span.
	if got := classOf(t, lines, " main"); got != Plain {
		t.Errorf("plain text class = %d", got)
	}
	// A raw string and a block comment span lines: both halves keep the class.
	if classOf(t, lines, "`raw") != String || classOf(t, lines, "line`") != String {
		t.Error("multi-line raw string not classed as String on both lines")
	}
	if classOf(t, lines, "/* a") != Comment || classOf(t, lines, "b */") != Comment {
		t.Error("multi-line block comment not classed as Comment on both lines")
	}
}

func TestJSONTokens(t *testing.T) {
	lines := Lines("json", `{"name": "x", "n": -12.5e3, "ok": true, "z": null, "a": ["s", 1]}`)
	for text, want := range map[string]Class{
		`"name"`: Key, `"n"`: Key, `"x"`: String, `"s"`: String,
		"-12.5e3": Number, "1": Number, "true": Literal, "null": Literal,
	} {
		if got := classOf(t, lines, text); got != want {
			t.Errorf("%q class = %d, want %d", text, got, want)
		}
	}
}

func TestShellTokens(t *testing.T) {
	lines := Lines("bash", "# top\nif [ -n $HOME ]; then echo ${X:-1} $1 # tail\nfi\necho a#b 'it''s'\n")
	for text, want := range map[string]Class{
		"# top": Comment, "if": Keyword, "then": Keyword, "fi": Keyword,
		"$HOME": Variable, "${X:-1}": Variable, "$1": Variable,
		"# tail": Comment, `'it''s'`: String,
	} {
		if got := classOf(t, lines, text); got != want {
			t.Errorf("%q class = %d, want %d", text, got, want)
		}
	}
	// '#' inside a word is not a comment.
	for _, l := range lines {
		for _, sp := range l {
			if strings.Contains(sp.Text, "a#b") && sp.Class == Comment {
				t.Errorf("a#b classed as Comment: %v", sp)
			}
		}
	}
}

// Lines always partitions the input: joining a line's spans gives that line.
func TestLinesPartitionTheInput(t *testing.T) {
	inputs := []string{
		"", "\n", "a", "a\n", "\n\n", "x := \"unterminated\nnext", "/* open", "`open raw\n",
		"é \"é\\é\" 日本語 // 語", "\\", "\"\\", "'", "$", "${", "${x",
	}
	for _, lang := range []string{"go", "json", "sh", "", "unknown"} {
		for _, in := range inputs {
			lines := Lines(lang, in)
			want := strings.Split(in, "\n")
			if len(lines) != len(want) {
				t.Fatalf("%s %q: %d lines, want %d", lang, in, len(lines), len(want))
			}
			for i, l := range lines {
				var b strings.Builder
				for _, sp := range l {
					if sp.Text == "" || strings.Contains(sp.Text, "\n") {
						t.Errorf("%s %q: bad span %q", lang, in, sp.Text)
					}
					if !utf8.ValidString(sp.Text) && utf8.ValidString(in) {
						t.Errorf("%s %q: span %q splits a rune", lang, in, sp.Text)
					}
					b.WriteString(sp.Text)
				}
				if b.String() != want[i] {
					t.Errorf("%s %q line %d = %q, want %q", lang, in, i, b.String(), want[i])
				}
			}
		}
	}
}

func FuzzLines(f *testing.F) {
	for _, s := range []string{"", "package p\nfunc f() {}", `{"a": [1, true]}`, "if x; then $y; fi # c", "`raw", "/*", "\"\\", "a: &x [1, 'b']\n- c: # d\n---\n", "def f(x):\n  return rb'a' + \"\"\"doc"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		for _, lang := range []string{"go", "json", "sh", "yaml", "python", "js", "ts", "rust", "c", "cpp", "java", "diff", "md", "sql", "toml", "dockerfile",
			"csharp", "kotlin", "swift", "php", "ruby", "lua", "html", "xml", "css", "ini", "makefile"} {
			lines := Lines(lang, in)
			var got []string
			for _, l := range lines {
				var b strings.Builder
				for _, sp := range l {
					b.WriteString(sp.Text)
				}
				got = append(got, b.String())
			}
			if strings.Join(got, "\n") != in {
				t.Fatalf("%s: spans do not reassemble the input %q", lang, in)
			}
		}
	})
}
