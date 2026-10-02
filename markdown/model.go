package markdown

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Model is a Markdown document held as a value, so it can take part in
// accessible output (Linearize) and layouts (LayoutNode) like the other
// widgets. It has no interaction state: Render and RenderWith remain the way
// to draw a string directly.
type Model struct {
	src  string
	opts Options
	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
}

// New returns a Model for the Markdown source md.
func New(md string) Model { return Model{src: md} }

// WithOptions returns m drawn with o (see RenderWith).
func (m Model) WithOptions(o Options) Model {
	m.opts = o
	return m
}

// Source returns the Markdown text m was built from.
func (m Model) Source() string { return m.src }

// View lays the document out in at most width columns with t, as RenderWith,
// after resolving t for theme.ComponentMarkdown and the tokens set by
// WithTokens. Render and RenderWith take the theme as given.
func (m Model) View(width int, t theme.Theme) string {
	return RenderWith(m.src, width, t.Resolve(theme.ComponentMarkdown, m.tokens), m.opts)
}

// Tokens returns the colour tokens the document renders with under t: t's
// roles, overridden by any theme.WithTokens(theme.ComponentMarkdown, ...) and
// then by WithTokens.
func (m Model) Tokens(t theme.Theme) theme.Tokens {
	return t.Resolve(theme.ComponentMarkdown, m.tokens).TokensFor("")
}

// WithTokens returns m with tok as its per-instance colour override. Nil
// fields inherit from the theme passed to View, so only the roles tok names
// change. A second call replaces the first.
func (m Model) WithTokens(tok theme.Tokens) Model {
	m.tokens = tok
	return m
}

// defaultMeasureWidth is the wrap width Measure assumes when its constraints
// set no maximum width.
const defaultMeasureWidth = 80

// LayoutNode adapts the document to a layout.Node drawn with theme t. Measure
// wraps at the constraint's maximum width (80 columns when unbounded) and
// reports the result's size; Render re-wraps at the allotted width and fits
// the allotted size exactly, clipping or padding as needed. The Model is not
// changed.
func (m Model) LayoutNode(t theme.Theme) layout.Node { return docNode{m, t} }

type docNode struct {
	m Model
	t theme.Theme
}

func (n docNode) Measure(c layout.Constraints) layout.Size {
	w := c.MaxW
	if w <= 0 || w == layout.Unbounded {
		w = defaultMeasureWidth
	}
	return layout.Block(n.m.View(w, n.t)).Measure(c)
}

func (n docNode) Render(s layout.Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	return layout.Block(n.m.View(s.W, n.t)).Render(s)
}

// Linearize renders the document as plain text for accessible output (see
// tui.Linearizer): no colour, wrapping, frames or markers. Headings read
// "Heading level N: text", list items keep their bullet or number, quotes
// read "Quote: ...", fenced code is introduced by "Code" (with its language)
// and closed by "End code", links read "text (url)", and tables read row by
// row as "header: cell" pairs. Rules are dropped. An empty document returns
// "Empty document".
func (m Model) Linearize() string {
	blocks := parseBlocks(splitLines(sanitize(m.src)), 0, 0)
	var out []string
	linearizeBlocks(blocks, "", &out)
	if len(out) == 0 {
		return "Empty document"
	}
	return strings.Join(out, "\n")
}

func inlineText(s string) string {
	return strings.Join(strings.Fields(plainText(parseInline(s))), " ")
}

func linearizeBlocks(blocks []block, prefix string, out *[]string) {
	for _, b := range blocks {
		switch b := b.(type) {
		case paragraph:
			*out = append(*out, prefix+inlineText(b.text))
		case heading:
			*out = append(*out, prefix+"Heading level "+strconv.Itoa(b.level)+": "+inlineText(b.text))
		case codeBlock:
			open := "Code"
			if b.lang != "" {
				open += " (" + b.lang + ")"
			}
			*out = append(*out, prefix+open)
			for _, l := range strings.Split(strings.TrimRight(b.text, "\n"), "\n") {
				*out = append(*out, prefix+l)
			}
			*out = append(*out, prefix+"End code")
		case table:
			for _, row := range b.rows {
				cells := make([]string, len(row))
				for i, c := range row {
					h := ""
					if i < len(b.header) {
						h = inlineText(b.header[i]) + ": "
					}
					cells[i] = h + inlineText(c)
				}
				*out = append(*out, prefix+strings.Join(cells, ", "))
			}
		case quote:
			*out = append(*out, prefix+"Quote:")
			linearizeBlocks(b.children, prefix+"  ", out)
		case list:
			for i, item := range b.items {
				mark := "- "
				if b.ordered {
					mark = strconv.Itoa(b.start+i) + ". "
				}
				var sub []string
				linearizeBlocks(item, "", &sub)
				for j, l := range sub {
					if j == 0 {
						l = mark + l
					} else {
						l = strings.Repeat(" ", len(mark)) + l
					}
					*out = append(*out, prefix+l)
				}
			}
		}
	}
}
