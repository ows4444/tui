package markdown

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/highlight"
	"github.com/ows4444/tui/theme"
)

// Class is the colour class of a Span. Plain is the zero value; the others
// map to theme colours the way widgets.CodeBlockLang does: Keyword to
// t.Primary, Type, Key and Variable to t.Info, String to t.Success, Number
// to t.Warning, Comment to t.Muted, Literal to t.Secondary and Plain to
// t.Text.
type Class int

const (
	// Plain is ordinary text and the zero value.
	Plain Class = iota
	// Keyword is a language keyword.
	Keyword
	// Type is a type name.
	Type
	// String is a string literal.
	String
	// Number is a numeric literal.
	Number
	// Comment is a comment.
	Comment
	// Literal is a value such as true, false, nil or null.
	Literal
	// Key is an object key.
	Key
	// Variable is a shell-style variable.
	Variable
)

// Span is a run of code text in one Class. It should not contain a newline;
// Render treats one as a line break it does not draw.
type Span struct {
	Text  string
	Class Class
}

// Lexer colours the code of a fenced block. Lex receives the code (tabs
// already expanded to four spaces, no trailing newline) and the fence's
// language, the first word after the opening fence and possibly empty. It
// returns the spans of each line, in order, and ok true; ok false means the
// Lexer does not handle lang and Render falls back to the built-in
// languages. The concatenated text of a line's spans should be that line;
// Render takes exactly one row per source line, padding a short result with
// plain text and ignoring extra lines.
//
// A Lexer is called once per fence, on the goroutine that called Render,
// and its output is sanitized, so it cannot inject escape sequences.
type Lexer interface {
	Lex(code, lang string) (lines [][]Span, ok bool)
}

// DefaultLexer returns the Lexer that highlights the languages the library ships with: "go",
// "json", "sh", "yaml" and "python" (and their usual aliases). It declines
// every other language. Render uses it when no Lexer handles a fence, so a
// caller's Lexer can call it to extend rather than replace the built-ins.
func DefaultLexer() Lexer { return builtinLexer{} }

type builtinLexer struct{}

func (builtinLexer) Lex(code, lang string) ([][]Span, bool) {
	if !highlight.Supported(lang) {
		return nil, false
	}
	hl := highlight.Lines(lang, code)
	out := make([][]Span, len(hl))
	for i, line := range hl {
		spans := make([]Span, len(line))
		for k, sp := range line {
			spans[k] = Span{Text: sp.Text, Class: Class(sp.Class)}
		}
		out[i] = spans
	}
	return out, true
}

// Class values mirror highlight.Class one for one; this fails to compile if
// they drift apart.
var _ = [1]struct{}{}[int(Variable)-int(highlight.Variable)]

func classColor(c Class, t theme.Theme) ansi.Color {
	switch c {
	case Keyword:
		return t.Primary
	case Type, Key, Variable:
		return t.Info
	case String:
		return t.Success
	case Number:
		return t.Warning
	case Comment:
		return t.Muted
	case Literal:
		return t.Secondary
	}
	return t.Text
}

// lexedLines runs lx on a fence and returns exactly one span row per source
// line, or ok false when lx declines. Span text is sanitized and stripped
// of newlines.
func lexedLines(lx Lexer, code, lang string) ([][]Span, bool) {
	if lx == nil {
		return nil, false
	}
	code = strings.TrimSuffix(strings.ReplaceAll(code, "\t", "    "), "\n")
	src := strings.Split(code, "\n")
	got, ok := lx.Lex(code, lang)
	if !ok {
		return nil, false
	}
	rows := make([][]Span, len(src))
	for i := range rows {
		if i >= len(got) {
			rows[i] = []Span{{Text: src[i]}}
			continue
		}
		for _, sp := range got[i] {
			text := strings.ReplaceAll(sanitize(sp.Text), "\n", "")
			text = strings.ReplaceAll(strings.ReplaceAll(text, "\r", ""), "\t", "    ")
			if text != "" {
				rows[i] = append(rows[i], Span{Text: text, Class: sp.Class})
			}
		}
	}
	return rows, true
}

// renderSpans draws span rows in the same box as widgets.CodeBlock: exactly
// width columns, a Muted ellipsis on a cut line, and no frame below
// codeFrameMinWidth columns.
func renderSpans(rows [][]Span, width int, t theme.Theme) []string {
	if width < codeFrameMinWidth {
		var out []string
		for _, r := range rows {
			if s, _ := cutSpans(r, width, t); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	inner := width - 4
	body := make([]string, len(rows))
	for i, r := range rows {
		body[i] = fitSpans(r, inner, t)
	}
	return frame(body, inner, t)
}

// cutSpans renders spans clipped to w columns and returns the text and its
// width.
func cutSpans(spans []Span, w int, t theme.Theme) (string, int) {
	var b strings.Builder
	used := 0
	for _, sp := range spans {
		if used >= w {
			break
		}
		text := ansi.Truncate(sp.Text, w-used)
		if text == "" {
			continue
		}
		b.WriteString(ansi.NewStyle().Foreground(classColor(sp.Class, t)).Render(text))
		used += ansi.Width(text)
	}
	return b.String(), used
}

// fitSpans pads or cuts spans to exactly w columns.
func fitSpans(spans []Span, w int, t theme.Theme) string {
	total := 0
	for _, sp := range spans {
		total += ansi.Width(sp.Text)
	}
	if total <= w {
		s, _ := cutSpans(spans, w, t)
		return s + strings.Repeat(" ", w-total)
	}
	s, used := cutSpans(spans, w-1, t)
	return s + ansi.NewStyle().Foreground(t.Muted).Render(t.GlyphSet().Ellipsis) + strings.Repeat(" ", w-1-used)
}

// frame draws t.Border around rows that are each inner columns wide, with
// one column of padding either side; empty border pieces become spaces.
func frame(rows []string, inner int, t theme.Theme) []string {
	b := t.Border
	edge := func(s string) string {
		if s == "" {
			return " "
		}
		return s
	}
	bs := ansi.NewStyle().Foreground(t.BorderColor)
	bar := func(fill, l, r string) string {
		return bs.Render(edge(l) + strings.Repeat(edge(fill), inner+2) + edge(r))
	}
	out := make([]string, 0, len(rows)+2)
	out = append(out, bar(b.Top, b.TopLeft, b.TopRight))
	left, right := bs.Render(edge(b.Left)), bs.Render(edge(b.Right))
	for _, r := range rows {
		out = append(out, left+" "+r+" "+right)
	}
	return append(out, bar(b.Bottom, b.BottomLeft, b.BottomRight))
}
