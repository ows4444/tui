package highlight

import (
	"strings"
	"testing"
)

func TestYAMLTokens(t *testing.T) {
	lines := Lines("yaml", "---\n# top\nname: \"web\"  # tail\nport: 8080\nratio: -1.5e3\non: true\nnone: null\nitems:\n  - id: 7\n  - 'x': &a text\n    ref: *a\n\"quoted key\": v\nversion: 1.2.3\nurl: http://x:80/a\n")
	for text, want := range map[string]Class{
		"---": Keyword, "# top": Comment, "# tail": Comment,
		"name": Key, `"web"`: String, "port": Key, "8080": Number, "-1.5e3": Number,
		"true": Literal, "null": Literal, "items": Key, "id": Key, "7": Number,
		"'x'": Key, "&a": Variable, "*a": Variable, `"quoted key"`: Key,
		"1.2.3": Number, " text": Plain, ": v": Plain,
	} {
		if got := classOf(t, lines, text); got != want {
			t.Errorf("%q class = %d, want %d", text, got, want)
		}
	}
	// A colon inside a value is not a key separator.
	for _, l := range lines {
		for _, sp := range l {
			if strings.Contains(sp.Text, "http") && sp.Class == Key {
				t.Errorf("URL classed as Key: %v", sp)
			}
		}
	}
}

func TestPythonTokens(t *testing.T) {
	lines := Lines("py", "import os  # c\n@dec\nclass A:\n    def f(self, n: int = 0x1F) -> None:\n        s = f'x{n}' + rb\"raw\" + \"\"\"doc\n        more\"\"\"\n        return True if n else 1_000.5e-3\n")
	for text, want := range map[string]Class{
		"import": Keyword, "# c": Comment, "class": Keyword, "def": Keyword,
		"int": Type, "0x1F": Number, "None": Literal, "True": Literal, "return": Keyword,
		"if": Keyword, "else": Keyword, "1_000.5e-3": Number, "f'x{n}'": String, `rb"raw"`: String,
	} {
		if got := classOf(t, lines, text); got != want {
			t.Errorf("%q class = %d, want %d", text, got, want)
		}
	}
	// A triple-quoted string keeps its class on the following line.
	found := false
	for _, sp := range lines[5] {
		if strings.Contains(sp.Text, `more"""`) && sp.Class == String {
			found = true
		}
	}
	if !found {
		t.Errorf("second line of the docstring is not a String: %v", lines[5])
	}
	// A '#' inside a string is not a comment.
	for _, sp := range Lines("python", "s = 'a # b'")[0] {
		if sp.Class == Comment {
			t.Errorf("# inside a string classed as Comment: %v", sp)
		}
	}
}

// Go, JSON and shell output is unchanged by the new languages.
func TestExistingLexersUnchanged(t *testing.T) {
	if got := classOf(t, Lines("json", `{"a": 1}`), `"a"`); got != Key {
		t.Errorf("json key class = %d", got)
	}
	if got := classOf(t, Lines("go", "func f() {}"), "func"); got != Keyword {
		t.Errorf("go keyword class = %d", got)
	}
	if got := classOf(t, Lines("sh", "echo $HOME"), "$HOME"); got != Variable {
		t.Errorf("sh variable class = %d", got)
	}
}

// Malformed and unterminated input never panics and reassembles exactly.
func TestNewLexersSurviveMalformedInput(t *testing.T) {
	inputs := []string{
		"a: 'open\nb: \"x", "k: [1, {a: *b}", "\"\"\"open\nx", "r'\\", "- - a: 1", "---", "a:", ": x",
		"'", "\"", "&", "!", "-", "- ", "a: - b", "\\", "\"\"\"", "'''a", "é: 日本\n", "x = 'é\\", "\t- \t#c",
	}
	for _, lang := range []string{"yaml", "python"} {
		for _, in := range inputs {
			var got []string
			for _, l := range Lines(lang, in) {
				var b strings.Builder
				for _, sp := range l {
					b.WriteString(sp.Text)
				}
				got = append(got, b.String())
			}
			if strings.Join(got, "\n") != in {
				t.Errorf("%s %q: spans reassemble to %q", lang, in, strings.Join(got, "\n"))
			}
		}
	}
}
