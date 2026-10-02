package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/highlight"
	"github.com/ows4444/tui/theme"
)

// CodeBlockLang is CodeBlock with syntax highlighting for lang: "go",
// "json", "sh", "yaml" or "python" (also "golang", "bash", "shell", "zsh",
// "yml", "py", "python3"); "js" ("javascript", "jsx", "mjs", "cjs"); "ts"
// ("typescript", "tsx"); "rust" ("rs"); "c" ("h"); "cpp" ("c++", "cc",
// "cxx", "hpp", "hh", "hxx"); "java"; "csharp" ("cs", "c#"); "kotlin"
// ("kt", "kts"); "swift"; "php"; "ruby" ("rb"); "lua"; "html" ("htm",
// "xhtml"; its <script> and <style> content is highlighted as JavaScript and
// CSS); "xml" ("svg", "xsd", "xsl", "plist"); "css"; "diff" ("patch");
// "markdown" ("md"); "sql"; "toml"; "ini" ("cfg", "conf", "properties",
// "editorconfig", "gitconfig"); "makefile" ("make", "mk", "gnumakefile");
// "dockerfile" ("docker"). Names are case-insensitive.
// Tokens are coloured from the theme: keywords in t.Primary, types in
// t.Info, strings in t.Success, numbers in t.Warning, comments in t.Muted,
// literals (true, nil, null, Ruby symbols) in t.Secondary, keys (JSON, YAML,
// TOML and INI keys, CSS properties, HTML and XML attributes) and variables
// (shell, PHP, Ruby and Makefile) in t.Info, and everything else in t.Text.
//
// The highlighter is a lexer, not a parser, so it colours what it
// recognises and never rejects input. For an empty or unsupported lang the
// result is byte-identical to CodeBlock. Layout, width, gutter, tab and
// truncation rules are exactly CodeBlock's: the output is always width
// columns wide and one row per source line plus the two border rows.
func CodeBlockLang(code, lang string, width int, lineNumbers bool, t theme.Theme) string {
	if !highlight.Supported(lang) {
		return CodeBlock(code, width, lineNumbers, t)
	}
	lines := codeLines(code)
	spans := highlight.Lines(lang, strings.Join(lines, "\n"))
	return renderCodeRows(len(lines), width, lineNumbers, t,
		func(i, w int) string { return narrowSpans(spans[i], w, t) },
		func(i, w int) string { return fitSpans(spans[i], w, t) })
}

// classColor maps a token class to a theme colour.
func classColor(c highlight.Class, t theme.Theme) ansi.Color {
	switch c {
	case highlight.Keyword:
		return t.Primary
	case highlight.Type, highlight.Key, highlight.Variable:
		return t.Info
	case highlight.String:
		return t.Success
	case highlight.Number:
		return t.Warning
	case highlight.Comment:
		return t.Muted
	case highlight.Literal:
		return t.Secondary
	}
	return t.Text
}

func renderSpan(text string, c highlight.Class, t theme.Theme) string {
	if text == "" {
		return ""
	}
	return ansi.NewStyle().Foreground(classColor(c, t)).Render(text)
}

// cutSpans renders the spans clipped to at most w columns and returns the
// text and its visible width.
func cutSpans(spans []highlight.Span, w int, t theme.Theme) (string, int) {
	var b strings.Builder
	used := 0
	for _, sp := range spans {
		if used >= w {
			break
		}
		text := ansi.Truncate(sp.Text, w-used)
		b.WriteString(renderSpan(text, sp.Class, t))
		used += ansi.Width(text)
	}
	return b.String(), used
}

// narrowSpans is the frameless row used below codeBlockMinWidth: the spans
// truncated to w columns.
func narrowSpans(spans []highlight.Span, w int, t theme.Theme) string {
	s, _ := cutSpans(spans, w, t)
	return s
}

// fitSpans returns the spans padded or cut to exactly w columns; a cut line
// ends in a Muted ellipsis, like fitCodeLine.
func fitSpans(spans []highlight.Span, w int, t theme.Theme) string {
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
