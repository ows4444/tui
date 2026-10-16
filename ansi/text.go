package ansi

import "strings"

// Span is a run of text in one Style. Its Text is plain text: escape sequences
// and control characters are not styling and are removed where a Text is built
// (NewText, Append), so a Span cannot smuggle terminal control codes into
// output.
type Span struct {
	Text  string
	Style Style
}

// Text is styled text kept as data: a list of spans. Unlike a styled string it
// can be measured, cut and wrapped without parsing escape sequences, and each
// piece keeps its style. A Text is a value; every method returns a new one and
// leaves the receiver alone. The zero value is empty. "\n" ends a line.
type Text []Span

// NewText returns a Text of one span. Tabs become spaces and escape sequences
// and other control characters (but not "\n") are removed. Empty text gives an
// empty Text.
func NewText(text string, style Style) Text { return Text(nil).Append(text, style) }

// Append returns t with text in style added at the end, cleaned as NewText does.
// A span in the same style as the last one is joined to it.
func (t Text) Append(text string, style Style) Text {
	text = Sanitize(ExpandTabs(text))
	if text == "" {
		return t
	}
	out := append(Text(nil), t...)
	if n := len(out); n > 0 && out[n-1].Style == style {
		out[n-1].Text += text
		return out
	}
	return append(out, Span{Text: text, Style: style})
}

// Plain returns the text without its styles.
func (t Text) Plain() string {
	var b strings.Builder
	for _, s := range t {
		b.WriteString(s.Text)
	}
	return b.String()
}

// String renders the text with its styles, one styled run per span (and per
// line, so every line is self-contained).
func (t Text) String() string {
	var out strings.Builder
	for i, line := range t.Lines() {
		if i > 0 {
			out.WriteByte('\n')
		}
		for _, s := range line {
			if s.Style == (Style{}) {
				out.WriteString(s.Text)
			} else {
				out.WriteString(s.Style.Render(s.Text))
			}
		}
	}
	return out.String()
}

// atom is one grapheme cluster with its column width and style.
type atom struct {
	s     string
	w     int
	style Style
}

// atoms splits t into lines of atoms.
func (t Text) atoms() [][]atom {
	lines := [][]atom{nil}
	for _, sp := range t {
		for _, c := range splitClusters(sp.Text) {
			if c == "\n" {
				lines = append(lines, nil)
				continue
			}
			n := len(lines) - 1
			lines[n] = append(lines[n], atom{s: c, w: Width(c), style: sp.Style})
		}
	}
	return lines
}

// fromAtoms joins lines of atoms into a Text, merging neighbours of one style.
func fromAtoms(lines [][]atom) Text {
	var out Text
	for i, line := range lines {
		if i > 0 {
			out = out.appendRaw("\n", Style{}, true)
		}
		for _, a := range line {
			out = out.appendRaw(a.s, a.style, false)
		}
	}
	return out
}

// appendRaw adds already-clean text; a newline span is kept apart so lines can
// be found again by splitting, never merged into a styled run.
func (t Text) appendRaw(s string, style Style, newline bool) Text {
	if n := len(t); n > 0 && !newline && t[n-1].Style == style && !strings.HasSuffix(t[n-1].Text, "\n") {
		t[n-1].Text += s
		return t
	}
	return append(t, Span{Text: s, Style: style})
}

// Lines splits the text at "\n" into one Text per line (an empty Text for an
// empty line). The text always has at least one line.
func (t Text) Lines() []Text {
	var out []Text
	for _, line := range t.atoms() {
		out = append(out, fromAtoms([][]atom{line}))
	}
	return out
}

// Width returns the display width of the widest line.
func (t Text) Width() int {
	w := 0
	for _, line := range t.atoms() {
		lw := 0
		for _, a := range line {
			lw += a.w
		}
		w = max(w, lw)
	}
	return w
}

// Height returns the number of lines: 1 for an empty Text.
func (t Text) Height() int { return len(t.atoms()) }

// Slice returns display columns [from, to) of each line, keeping each piece's
// style. A wide character that straddles an edge is left out, so the result
// never holds half of one.
func (t Text) Slice(from, to int) Text {
	lines := t.atoms()
	for i, line := range lines {
		var kept []atom
		col := 0
		for _, a := range line {
			if col >= from && col+a.w <= to {
				kept = append(kept, a)
			}
			col += a.w
		}
		lines[i] = kept
	}
	return fromAtoms(lines)
}

// Truncate cuts every line to width columns. A wide character that would cross
// the edge is left out.
func (t Text) Truncate(width int) Text { return t.Slice(0, max(width, 0)) }

// Wrap breaks the text into lines of at most width columns, at spaces. Existing
// newlines are kept; the spaces where a line is broken are dropped; a word wider
// than width is cut at a character boundary rather than left to overflow. Each
// piece keeps its style. A width under 1 gives one line per input line,
// unchanged.
func (t Text) Wrap(width int) []Text {
	paragraphs := t.atoms()
	if width < 1 {
		out := make([]Text, len(paragraphs))
		for i, p := range paragraphs {
			out[i] = fromAtoms([][]atom{p})
		}
		return out
	}
	var out []Text
	for _, p := range paragraphs {
		for _, line := range wrapAtoms(p, width) {
			out = append(out, fromAtoms([][]atom{line}))
		}
	}
	return out
}

// wrapAtoms breaks one paragraph into lines.
func wrapAtoms(p []atom, width int) [][]atom {
	if len(p) == 0 {
		return [][]atom{nil}
	}
	var lines [][]atom
	var cur []atom
	curW := 0
	flush := func() {
		lines = append(lines, cur)
		cur, curW = nil, 0
	}
	isSpace := func(a atom) bool { return a.s == " " }
	for i := 0; i < len(p); {
		j := i
		for j < len(p) && isSpace(p[j]) == isSpace(p[i]) {
			j++
		}
		run := p[i:j]
		runW := 0
		for _, a := range run {
			runW += a.w
		}
		switch {
		case isSpace(run[0]):
			// Spaces are kept inside a line, and at its very start (an indent);
			// where they would end or begin a broken line they are dropped.
			if curW+runW <= width && (len(cur) > 0 || len(lines) == 0) {
				cur = append(cur, run...)
				curW += runW
			} else if len(cur) > 0 && curW+runW > width {
				// they would overflow: break here and drop them
				flush()
			}
		case curW+runW <= width:
			cur = append(cur, run...)
			curW += runW
		default:
			if len(cur) > 0 && trailingSpaces(cur) {
				cur = trimTrailingSpaces(cur)
			}
			if len(cur) > 0 {
				flush()
			}
			// A word wider than the line is cut into line-sized pieces.
			for _, a := range run {
				if curW+a.w > width && len(cur) > 0 {
					flush()
				}
				cur = append(cur, a)
				curW += a.w
			}
		}
		i = j
	}
	if len(cur) > 0 || len(lines) == 0 {
		cur = trimTrailingSpaces(cur)
		lines = append(lines, cur)
	}
	return lines
}

func trailingSpaces(a []atom) bool { return len(a) > 0 && a[len(a)-1].s == " " }

func trimTrailingSpaces(a []atom) []atom {
	for len(a) > 0 && a[len(a)-1].s == " " {
		a = a[:len(a)-1]
	}
	return a
}

// WrapString renders the text wrapped to width columns (see Wrap) as one styled
// string, lines joined with "\n". A width under 1 renders it unwrapped, as
// String does.
func (t Text) WrapString(width int) string {
	if width < 1 {
		return t.String()
	}
	lines := t.Wrap(width)
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.String()
	}
	return strings.Join(out, "\n")
}

// WithDefault returns t with style given to every span that has none, so text
// built without styles takes a surrounding colour while spans that set their own
// keep it.
func (t Text) WithDefault(style Style) Text {
	out := make(Text, len(t))
	for i, sp := range t {
		if sp.Style == (Style{}) {
			sp.Style = style
		}
		out[i] = sp
	}
	return out
}
