// Package markdown renders a subset of CommonMark as styled, width-aware
// terminal text, written from scratch with no dependencies.
//
// Supported: ATX and setext headings, paragraphs (soft and hard breaks),
// emphasis and strong, inline code, GFM pipe tables (aligned columns,
// wrapped to fit), fenced code blocks (drawn with widgets.CodeBlock, and
// coloured by a Lexer passed to RenderWith), bullet and ordered lists with
// one level of nesting, block quotes, horizontal rules, and inline links and
// images (shown as "text (url)").
//
// Everything else renders as plain paragraph text: HTML, reference links,
// footnotes, indented code and entities. Deliberate departures
// from CommonMark: block quotes have no lazy continuation, list items are
// always tight (only two paragraphs in one item are separated by a blank
// line), list markers deeper than one nested level are text, and an item's
// content may start up to four spaces after its marker.
//
// Styling is applied per word, never across a line break, so every output
// line is self-contained and safe to repaint on its own (see Program.render).
package markdown

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// codeFrameMinWidth mirrors the width below which widgets.CodeBlock stops
// drawing its border.
const codeFrameMinWidth = 5

// Render lays md out in at most width columns using t's colours: headings
// in Primary (level 1 underlined), inline code in Secondary, links in Info
// with the url Muted after them, list markers in Primary, quote bars and
// rules in Muted, and body text in Text. Paragraphs wrap on word
// boundaries; a word longer than the line is broken by column, so no line
// is ever wider than width. Fenced code is boxed at the available width.
//
// Escape sequences and control characters in md are stripped first, so
// untrusted input can't inject terminal control codes. A width of zero or
// less, or empty input, returns "". The result has no leading or trailing
// blank line and blocks are separated by exactly one.
func Render(md string, width int, t theme.Theme) string {
	return RenderWith(md, width, t, Options{})
}

// Options tunes RenderWith. The zero value is Render's behaviour.
type Options struct {
	// Lexer colours fenced code blocks. Where it is nil or declines a
	// fence's language the fence is drawn plain, exactly as Render does;
	// pass DefaultLexer() (or a Lexer that falls back to it) to colour the
	// built-in languages.
	Lexer Lexer
}

// RenderWith is Render with o; see Options.
func RenderWith(md string, width int, t theme.Theme, o Options) string {
	if width <= 0 {
		return ""
	}
	builtin, cached := cacheable(md, o)
	key := cacheKey{md, width, t, builtin}
	if cached {
		if s, ok := cacheGet(key); ok {
			return s
		}
	}
	res := renderUncached(md, width, t, o)
	if cached {
		cachePut(key, res)
	}
	return res
}

func renderUncached(md string, width int, t theme.Theme, o Options) string {
	lines := splitLines(sanitize(md))
	out := renderBlocks(parseBlocks(lines, 0, 0), width, t, 0, o)
	for i, l := range out {
		if ansi.Width(l) > width { // only reachable at tiny widths
			out[i] = ansi.Truncate(l, width)
		}
	}
	return strings.Join(out, "\n")
}

// sanitize drops escape sequences and control characters other than
// newline, tab and carriage return (CR is normalised by splitLines).
func sanitize(s string) string {
	s = strings.ToValidUTF8(s, replacementRune)
	if !strings.ContainsAny(s, "\t\r") {
		return ansi.Sanitize(s)
	}
	// Keep tab and CR through the shared helper by parking them on two
	// noncharacters (any real ones become the replacement rune first).
	s = strings.NewReplacer(parkTab, replacementRune, parkCR, replacementRune).Replace(s)
	s = strings.NewReplacer("\t", parkTab, "\r", parkCR).Replace(s)
	s = ansi.Sanitize(s)
	return strings.NewReplacer(parkTab, "\t", parkCR, "\r").Replace(s)
}

// Built from rune values, not literals, so this file holds no glyph literals.
var (
	replacementRune = string(rune(0xFFFD))
	parkTab         = string(rune(0xFDD0))
	parkCR          = string(rune(0xFDD1))
)

// splitLines normalises line endings and turns each leading tab into four
// spaces so indentation can be measured in columns.
func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		n := 0
		for n < len(l) && (l[n] == ' ' || l[n] == '\t') {
			n++
		}
		if strings.IndexByte(l[:n], '\t') >= 0 {
			lead := strings.ReplaceAll(l[:n], "\t", "    ")
			lines[i] = lead + l[n:]
		}
	}
	return lines
}

// look is the base styling of a block's text; inline attrs add to it.
type look struct {
	fg        ansi.Color
	hasFG     bool
	bold      bool
	underline bool
	// typo, when hasTypo, is the theme's Typography style for this block
	// (H1, H2); it replaces fg and underline.
	typo    ansi.Style
	hasTypo bool
}

func (l look) style(a attrs, t theme.Theme) ansi.Style {
	ty := t.ResolvedTypography()
	var s ansi.Style
	switch {
	case a.muted:
		s = ansi.NewStyle().Foreground(t.Muted)
	case a.link:
		s = ty.Link
	case a.code:
		s = ty.Code
	case l.hasTypo:
		s = l.typo
	default:
		fg := t.Text
		if l.hasFG {
			fg = l.fg
		}
		s = ansi.NewStyle().Foreground(fg)
		if l.underline {
			s = s.Underline()
		}
	}
	if l.bold || a.bold {
		s = s.Bold()
	}
	if a.italic {
		s = s.Italic()
	}
	return s
}

func headingLook(level int, t theme.Theme) look {
	ty := t.ResolvedTypography()
	switch level {
	case 1:
		return look{typo: ty.H1, hasTypo: true}
	case 2:
		return look{typo: ty.H2, hasTypo: true}
	default:
		return look{bold: true}
	}
}

// renderBlocks renders blocks separated by one blank line; blocks that
// produce no lines (an empty heading) leave no gap.
func renderBlocks(blocks []block, avail int, t theme.Theme, depth int, o Options) []string {
	var out []string
	for _, b := range blocks {
		lines := renderBlock(b, avail, t, depth, o)
		if allBlank(lines) {
			continue
		}
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, lines...)
	}
	return out
}

// allBlank reports whether lines draw nothing: no lines, or only empty ones
// (an empty code block squeezed below CodeBlock's minimum width).
func allBlank(lines []string) bool {
	for _, l := range lines {
		if l != "" {
			return false
		}
	}
	return true
}

func renderBlock(b block, avail int, t theme.Theme, depth int, o Options) []string {
	switch b := b.(type) {
	case paragraph:
		return wrapPieces(parseInline(b.text), avail, look{}, t)
	case heading:
		return wrapPieces(parseInline(b.text), avail, headingLook(b.level, t), t)
	case table:
		return renderTable(b, avail, t)
	case codeBlock:
		if rows, ok := lexedLines(o.Lexer, b.text, b.lang); ok {
			return renderSpans(rows, avail, t)
		}
		lines := strings.Split(widgets.CodeBlock(b.text, avail, false, t), "\n")
		if avail < codeFrameMinWidth {
			// CodeBlock drops its frame this narrow, so blank code lines
			// would show as gaps in the output; drop them too.
			kept := lines[:0]
			for _, l := range lines {
				if l != "" {
					kept = append(kept, l)
				}
			}
			lines = kept
		}
		return lines
	case rule:
		return []string{ansi.NewStyle().Foreground(t.Muted).Render(strings.Repeat(t.GlyphSet().RuleH, avail))}
	case quote:
		return renderQuote(b, avail, t, depth, o)
	case list:
		return renderList(b, avail, t, depth, o)
	}
	return nil
}

func renderQuote(q quote, avail int, t theme.Theme, depth int, o Options) []string {
	bar := ansi.NewStyle().Foreground(t.Muted).Render(t.GlyphSet().RuleV)
	kids := renderBlocks(q.children, max(avail-2, 1), t, depth, o)
	if len(kids) == 0 {
		return []string{bar}
	}
	out := make([]string, len(kids))
	for i, k := range kids {
		if k == "" {
			out[i] = bar
		} else {
			out[i] = bar + " " + k
		}
	}
	return out
}

func renderList(l list, avail int, t theme.Theme, depth int, o Options) []string {
	marker := ansi.NewStyle().Foreground(t.Primary)
	numW := len(strconv.Itoa(l.start + len(l.items) - 1))

	var out []string
	for i, item := range l.items {
		var text string
		switch {
		case l.ordered:
			text = fmt.Sprintf("%*d%c ", numW, l.start+i, l.delim)
		case depth == 0:
			text = t.GlyphSet().Bullet + " "
		default:
			text = t.GlyphSet().BulletNested + " "
		}
		mw := ansi.Width(text)

		kids := renderItem(item, max(avail-mw, 1), t, depth+1, o)
		if len(kids) == 0 {
			out = append(out, marker.Render(strings.TrimRight(text, " ")))
			continue
		}
		pad := strings.Repeat(" ", mw)
		for j, k := range kids {
			switch {
			case j == 0:
				out = append(out, marker.Render(text)+k)
			case k == "":
				out = append(out, "")
			default:
				out = append(out, pad+k)
			}
		}
	}
	return out
}

// renderItem renders a list item's blocks tightly — no blank line between
// them — except between two paragraphs, which would otherwise run together.
func renderItem(blocks []block, avail int, t theme.Theme, depth int, o Options) []string {
	var out []string
	var prev block
	for _, b := range blocks {
		lines := renderBlock(b, avail, t, depth, o)
		if allBlank(lines) {
			continue
		}
		_, prevPara := prev.(paragraph)
		_, curPara := b.(paragraph)
		if prevPara && curPara {
			out = append(out, "")
		}
		out = append(out, lines...)
		prev = b
	}
	return out
}

// --- inline layout ---

type tokKind int

const (
	tokWord tokKind = iota
	tokSpace
	tokBreak
)

// token is a breakable unit of an inline run: a word (possibly several
// differently styled pieces, as in "foo**bar**"), a space, or a hard break.
type token struct {
	kind   tokKind
	pieces []piece
}

func isWS(c byte) bool { return c == ' ' || c == '\t' || c == '\n' }

func tokenizeWords(ps []piece) []token {
	// Exact pre-count so the buffers are right-sized.
	nTok, nAll := 1, 0
	for _, p := range ps {
		switch {
		case p.brk:
			nTok++
		case p.a.code:
			if p.text != "" {
				nAll++
			}
		default:
			for s := p.text; s != ""; {
				sp := isWS(s[0])
				i := 0
				for i < len(s) && isWS(s[i]) == sp {
					i++
				}
				if sp {
					nTok++
				} else {
					nAll++
				}
				s = s[i:]
			}
		}
	}
	toks := make([]token, 0, nTok+nAll)
	// Word pieces share one backing array; each token gets a capped
	// sub-slice of it.
	all := make([]piece, 0, nAll)
	start := 0
	flush := func() {
		if len(all) > start {
			toks = append(toks, token{kind: tokWord, pieces: all[start:len(all):len(all)]})
			start = len(all)
		}
	}
	for _, p := range ps {
		switch {
		case p.brk:
			flush()
			toks = append(toks, token{kind: tokBreak})
		case p.a.code:
			// A code span is one unbreakable word, inner spaces included.
			if p.text != "" {
				all = append(all, p)
			}
		default:
			for s := p.text; s != ""; {
				sp := isWS(s[0])
				i := 0
				for i < len(s) && isWS(s[i]) == sp {
					i++
				}
				if sp {
					flush()
					toks = append(toks, token{kind: tokSpace})
				} else {
					all = append(all, piece{text: s[:i], a: p.a})
				}
				s = s[i:]
			}
		}
	}
	flush()
	return toks
}

func wordWidth(ps []piece) int {
	w := 0
	for _, p := range ps {
		w += ansi.Width(p.text)
	}
	return w
}

// wrapPieces word-wraps pieces to avail columns. Each piece is styled on
// its own, so no line ends with an open style; a word wider than a line is
// split by column (a wide rune is never cut).
func wrapPieces(ps []piece, avail int, base look, t theme.Theme) []string {
	avail = max(avail, 1)

	lines := make([]string, 0, 4)
	cur := make([]byte, 0, 512) // reused across lines; each line is copied out once
	curW := 0
	has := false
	pendingSpace := false

	endLine := func() {
		lines = append(lines, string(cur))
		cur = cur[:0]
		curW, has, pendingSpace = 0, false, false
	}
	// Escape prefixes per distinct attrs, derived from Render itself so the
	// bytes stay identical without allocating a string per word.
	type prefix struct {
		a      attrs
		open   string
		styled bool
	}
	prefixes := make([]prefix, 0, 4)
	write := func(text string, a attrs) {
		if text == "" {
			return
		}
		if strings.IndexByte(text, '\n') >= 0 {
			cur = append(cur, base.style(a, t).Render(text)...)
			return
		}
		var p *prefix
		for i := range prefixes {
			if prefixes[i].a == a {
				p = &prefixes[i]
				break
			}
		}
		if p == nil {
			probe := base.style(a, t).Render("x")
			np := prefix{a: a}
			if probe != "x" {
				np.open = probe[:len(probe)-len(ansi.Reset)-1]
				np.styled = true
			}
			prefixes = append(prefixes, np)
			p = &prefixes[len(prefixes)-1]
		}
		if !p.styled {
			cur = append(cur, text...)
			return
		}
		cur = append(cur, p.open...)
		cur = append(cur, text...)
		cur = append(cur, ansi.Reset...)
	}

	for _, tk := range tokenizeWords(ps) {
		switch tk.kind {
		case tokSpace:
			if has {
				pendingSpace = true
			}
		case tokBreak:
			if has {
				endLine()
			}
		case tokWord:
			w := wordWidth(tk.pieces)
			need := w
			if pendingSpace {
				need++
			}
			if has && curW+need > avail {
				endLine()
			}
			if has && pendingSpace {
				cur = append(cur, ' ')
				curW++
			}
			pendingSpace = false

			if curW+w <= avail {
				for _, p := range tk.pieces {
					write(p.text, p.a)
				}
				curW += w
				has = true
				continue
			}
			// Too wide even for an empty line: break it by column.
			for _, p := range tk.pieces {
				var run strings.Builder
				for _, r := range p.text {
					rw := ansi.Width(string(r))
					if curW+rw > avail && curW > 0 {
						write(run.String(), p.a)
						run.Reset()
						endLine()
					}
					run.WriteRune(r)
					curW += rw
					has = true
				}
				write(run.String(), p.a)
			}
		}
	}
	if has {
		endLine()
	}
	return lines
}
