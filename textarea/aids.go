package textarea

import (
	"github.com/ows4444/tui/ansi"
)

// Editor aids: a highlighted cursor line, matching brackets, visible spaces and
// vertical rulers. They are drawn over the text by the same per-cluster path a
// selection or a Highlighter uses, so they work with both, with soft wrap, the
// horizontal window and the Height window.

const (
	// bracketScanLimit is how far, in runes, the search for a matching bracket
	// goes before it gives up, so a huge value cannot make a frame slow.
	bracketScanLimit = 20000

	defaultWhitespaceMark = "·"
)

var (
	defaultCursorLine ansi.Color = ansi.Color256(236)
	defaultRuler      ansi.Color = ansi.Color256(238)
)

// aidsOn reports whether any editor aid is set.
func (m Model) aidsOn() bool {
	return m.HighlightCursorLine || m.MatchBrackets || m.ShowWhitespace || len(m.Ruler) > 0
}

// aids is what a frame needs to know to draw the aids: where the cursor line
// is and which two runes are matching brackets.
type aids struct {
	m          *Model
	curLine    int
	pair       [2]int
	hasPair    bool
	cursorLine bool // the row being drawn is on the cursor line
}

func (m *Model) newAids() aids {
	a := aids{m: m, curLine: m.value.lineOf(m.cursor)}
	if m.MatchBrackets {
		a.pair[0], a.pair[1], a.hasPair = m.bracketPair()
	}
	return a
}

// forRow returns a for the row r.
func (a aids) forRow(r srow) aids {
	a.cursorLine = a.m.HighlightCursorLine && a.m.value.lineOf(r.base) == a.curLine
	return a
}

func (m Model) cursorLineColor() ansi.Color {
	if m.CursorLineColor != nil {
		return m.CursorLineColor
	}
	return defaultCursorLine
}

func (m Model) rulerColor() ansi.Color {
	if m.RulerColor != nil {
		return m.RulerColor
	}
	return defaultRuler
}

func (m Model) whitespaceMark() string {
	if m.WhitespaceMark != "" {
		return ansi.Sanitize(m.WhitespaceMark)
	}
	return defaultWhitespaceMark
}

// onRuler reports whether the cell run of w columns at column x covers a ruler.
func (m Model) onRuler(x, w int) bool {
	for _, col := range m.Ruler {
		if col >= 1 && col-1 >= x && col-1 < x+max(w, 1) {
			return true
		}
	}
	return false
}

// apply returns the style and text to draw for the text cluster c at rune
// offset pos, which starts at column x of the row, given its base style st.
func (a aids) apply(st ansi.Style, c string, pos, x int) (ansi.Style, string) {
	m := a.m
	if a.cursorLine {
		st = st.Background(m.cursorLineColor())
	}
	if m.onRuler(x, ansi.Width(c)) {
		st = st.Background(m.rulerColor())
	}
	if a.hasPair && (pos == a.pair[0] || pos == a.pair[1]) {
		st = st.Reverse().Bold()
	}
	if m.ShowWhitespace && c == " " {
		return st.Faint(), m.whitespaceMark()
	}
	return st, c
}

// tail calls f for the cells after the last cluster of a row that ends at
// column x: spaces up to the widest ruler, and on the cursor line up to Width.
func (a aids) tail(x int, base ansi.Style, f func(text string, st ansi.Style)) {
	m := a.m
	end := x
	for _, col := range m.Ruler {
		end = max(end, col)
	}
	if a.cursorLine && m.Width > 0 {
		end = max(end, m.Width)
	}
	for ; x < end; x++ {
		st := base
		if a.cursorLine {
			st = st.Background(m.cursorLineColor())
		}
		if m.onRuler(x, 1) {
			st = st.Background(m.rulerColor())
		}
		f(" ", st)
	}
}

// bracketPair finds the bracket under the cursor, or failing that the one just
// before it, and the bracket that matches it. It does not know about strings or
// comments, and gives up after bracketScanLimit runes.
func (m Model) bracketPair() (a, b int, ok bool) {
	for _, at := range []int{m.cursor, m.cursor - 1} {
		if at < 0 || at >= m.value.len() {
			continue
		}
		open, closing, forward := bracketKind(m.value.at(at))
		if open == 0 {
			continue
		}
		depth := 0
		for i, n := at, 0; i >= 0 && i < m.value.len() && n < bracketScanLimit; n++ {
			switch m.value.at(i) {
			case open:
				if forward {
					depth++
				} else {
					depth--
				}
			case closing:
				if forward {
					depth--
				} else {
					depth++
				}
			}
			if depth == 0 {
				return at, i, true
			}
			if forward {
				i++
			} else {
				i--
			}
		}
	}
	return 0, 0, false
}

// bracketKind says which bracket pair r belongs to and whether r opens it.
func bracketKind(r rune) (open, closing rune, forward bool) {
	switch r {
	case '(':
		return '(', ')', true
	case ')':
		return '(', ')', false
	case '[':
		return '[', ']', true
	case ']':
		return '[', ']', false
	case '{':
		return '{', '}', true
	case '}':
		return '{', '}', false
	}
	return 0, 0, false
}
