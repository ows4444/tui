// Package highlight splits source code into classed spans for a set of
// languages (Go, JSON, shell, YAML, Python, JavaScript, TypeScript, Rust, C,
// C++, Java, C#, Kotlin, Swift, PHP, Ruby, Lua, HTML, XML, CSS, diff,
// Markdown, SQL, TOML, INI, Makefile, Dockerfile). It is a lexer, not a parser: it recognises
// comments, strings, numbers, keywords and a few other token classes, which is
// enough to colour a code listing. Standard library only.
package highlight

import "strings"

// Class is a token class. Plain is the zero value.
type Class int

const (
	Plain Class = iota
	Keyword
	Type
	String
	Number
	Comment
	Literal  // true, false, nil, null, iota
	Key      // a JSON object key
	Variable // a shell variable such as $HOME
)

// Span is a run of text of one class. A span never contains a newline.
type Span struct {
	Text  string
	Class Class
}

// Supported reports whether lang names a language Lines can highlight.
func Supported(lang string) bool { return lexerFor(lang) != nil }

// Lines highlights code in lang and returns its spans grouped by line, split
// at every "\n" (one entry per line, so len == strings.Count(code, "\n")+1).
// The concatenated text of a line's spans is exactly that line. For an
// unsupported language every line is a single Plain span.
func Lines(lang, code string) [][]Span {
	lex := lexerFor(lang)
	var toks []Span
	if lex == nil {
		toks = []Span{{Text: code}}
	} else {
		toks = lex(code)
	}
	return splitLines(toks)
}

type lexer func(code string) []Span

func lexerFor(lang string) lexer {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "go", "golang":
		return lexGo
	case "json":
		return lexJSON
	case "sh", "bash", "shell", "zsh":
		return lexShell
	case "yaml", "yml":
		return lexYAML
	case "python", "py", "python3":
		return lexPython
	case "js", "javascript", "jsx", "mjs", "cjs":
		return clikeLexer("js")
	case "ts", "typescript", "tsx":
		return clikeLexer("ts")
	case "rs", "rust":
		return clikeLexer("rust")
	case "c", "h":
		return clikeLexer("c")
	case "cpp", "c++", "cc", "cxx", "hpp", "hh", "hxx":
		return clikeLexer("cpp")
	case "java":
		return clikeLexer("java")
	case "diff", "patch":
		return lexDiff
	case "md", "markdown":
		return lexMarkdown
	case "sql":
		return lexSQL
	case "toml":
		return lexTOML
	case "dockerfile", "docker":
		return lexDockerfile
	case "cs", "c#", "csharp":
		return clikeLexer("csharp")
	case "kt", "kts", "kotlin":
		return clikeLexer("kotlin")
	case "swift":
		return clikeLexer("swift")
	case "php":
		return clikeLexer("php")
	case "rb", "ruby":
		return lexRuby
	case "lua":
		return lexLua
	case "html", "htm", "xhtml":
		return lexHTML
	case "xml", "svg", "xsd", "xsl", "plist":
		return lexXML
	case "css":
		return lexCSS
	case "ini", "cfg", "conf", "properties", "editorconfig", "gitconfig":
		return lexINI
	case "make", "makefile", "mk", "gnumakefile":
		return lexMakefile
	}
	return nil
}

// splitLines cuts spans at newlines, dropping the newlines themselves.
func splitLines(toks []Span) [][]Span {
	lines := [][]Span{nil}
	for _, t := range toks {
		parts := strings.Split(t.Text, "\n")
		for i, p := range parts {
			if i > 0 {
				lines = append(lines, nil)
			}
			if p != "" {
				n := len(lines) - 1
				lines[n] = append(lines[n], Span{Text: p, Class: t.Class})
			}
		}
	}
	return lines
}

// emitter accumulates spans, merging neighbours of the same class. A span
// grown by merges collects its text in tail, so adding a long run one byte at
// a time stays linear instead of copying the span on every add.
type emitter struct {
	spans []Span
	tail  []byte // the last span's text while merges grow it; nil otherwise
}

func (e *emitter) add(text string, c Class) {
	if text == "" {
		return
	}
	if n := len(e.spans); n > 0 && e.spans[n-1].Class == c {
		if e.tail == nil {
			e.tail = append(make([]byte, 0, 2*(len(e.spans[n-1].Text)+len(text))), e.spans[n-1].Text...)
		}
		e.tail = append(e.tail, text...)
		return
	}
	e.flush()
	e.spans = append(e.spans, Span{Text: text, Class: c})
}

// flush writes a merged tail back into the last span.
func (e *emitter) flush() {
	if e.tail != nil {
		e.spans[len(e.spans)-1].Text = string(e.tail)
		e.tail = nil
	}
}

// result returns the spans, with any pending merge written back.
func (e *emitter) result() []Span {
	e.flush()
	return e.spans
}

func isIdentStart(b byte) bool {
	return b == '_' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= 0x80
}

func isIdentPart(b byte) bool { return isIdentStart(b) || b >= '0' && b <= '9' }

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

// scanQuoted returns the end index (exclusive) of a quoted string starting at
// s[i] (the opening quote). A backslash escapes the next byte. The string ends
// at the closing quote, or at a newline / the end of input if unterminated,
// unless multiline is set.
func scanQuoted(s string, i int, quote byte, multiline bool) int {
	j := i + 1
	for j < len(s) {
		switch c := s[j]; {
		case c == '\\' && quote != '`' && j+1 < len(s):
			j += 2
			for j < len(s) && s[j]&0xC0 == 0x80 { // keep a multi-byte rune whole
				j++
			}
			continue
		case c == quote:
			return j + 1
		case c == '\n' && !multiline:
			return j
		}
		j++
	}
	return j
}

// scanNumber returns the end of a numeric literal starting at s[i].
func scanNumber(s string, i int) int {
	j := i
	for j < len(s) {
		c := s[j]
		if isIdentPart(c) || c == '.' {
			j++
			continue
		}
		// An exponent sign: 1e-3, 0x1p+2.
		if (c == '+' || c == '-') && j > i && (s[j-1] == 'e' || s[j-1] == 'E' || s[j-1] == 'p' || s[j-1] == 'P') {
			j++
			continue
		}
		break
	}
	return j
}
