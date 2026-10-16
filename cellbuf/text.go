package cellbuf

import "github.com/ows4444/tui/ansi"

// SetText writes the first line of t on row y starting at column x, each span
// in its own style, and returns the number of columns it wrote. It does not
// wrap or parse anything: the spans are written as they are, so drawing styled
// text costs no escape-sequence parse. Text past the right edge, or above or
// left of b, is clipped as SetString clips it; a span's style the grid cannot
// hold is drawn in the default style.
func (b *Buffer) SetText(x, y int, t ansi.Text) int {
	wrote := 0
	for _, sp := range t {
		if i := indexNewline(sp.Text); i >= 0 {
			sp.Text = sp.Text[:i]
			wrote += b.setSpan(x+wrote, y, sp)
			break
		}
		wrote += b.setSpan(x+wrote, y, sp)
	}
	return wrote
}

func (b *Buffer) setSpan(x, y int, sp ansi.Span) int {
	var id StyleID
	if sp.Style != (ansi.Style{}) {
		if p, err := Parse(sp.Style.Render("x")); err == nil {
			id = b.StyleID(p.Style(p.At(0, 0).Style))
		}
	}
	return b.SetString(x, y, sp.Text, id)
}

func indexNewline(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return i
		}
	}
	return -1
}
