package viewport

import (
	"sort"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/keymap"
)

const (
	hlOn    = "\x1b[7m"   // reverse video: every match
	hlCur   = "\x1b[1;7m" // bold reverse: the current match
	hlOff   = "\x1b[27m"
	hlCurOf = "\x1b[22;27m"
)

// matchRef locates one match: line is an index into Model.lines, start the
// byte offset in the line's plain text, c0/c1 its display column range.
type matchRef struct {
	line, start, c0, c1 int
}

func defaultNextMatch() keymap.Binding { return keymap.NewBinding("next match", "n") }
func defaultPrevMatch() keymap.Binding { return keymap.NewBinding("previous match", "N") }

// spans returns the matches of q (lq is its lowered form) in line, in order.
// Matching is case-insensitive and runs on the text with styling removed.
func spans(line, q, lq string) []matchRef {
	if lq == "" {
		return nil
	}
	p := line
	if _, ok := ansi.PlainASCIIWidth(line); !ok {
		p = ansi.StripANSI(line)
	}
	lp, needle := strings.ToLower(p), lq
	if len(lp) != len(p) {
		lp, needle = p, q
	}
	var out []matchRef
	pos, col := 0, 0
	for {
		i := strings.Index(lp[pos:], needle)
		if i < 0 {
			return out
		}
		start := pos + i
		c0 := col + ansi.Width(p[pos:start])
		c1 := c0 + ansi.Width(p[start:start+len(needle)])
		out = append(out, matchRef{start: start, c0: c0, c1: c1})
		pos, col = start+len(needle), c1
	}
}

// matchesOf returns the matches in lines[idx] with line filled in.
func (m Model) matchesOf(idx int) []matchRef {
	ms := spans(m.lines[idx], m.query, strings.ToLower(m.query))
	for i := range ms {
		ms[i].line = idx
	}
	return ms
}

func (m Model) allMatches() []matchRef {
	if m.query == "" {
		return nil
	}
	lq := strings.ToLower(m.query)
	var out []matchRef
	for i := m.head; i < len(m.lines); i++ {
		for _, s := range spans(m.lines[i], m.query, lq) {
			s.line = i
			out = append(out, s)
		}
	}
	return out
}

// highlight reverses the display-column ranges of s (sorted, disjoint,
// relative to s) and bolds the one at index cur.
func highlight(s string, rs []matchRef, cur int) string {
	var b strings.Builder
	rest, done := s, 0
	for i, r := range rs {
		pre := r.c0 - done
		b.WriteString(ansi.Truncate(rest, pre))
		mid := ansi.StripANSI(ansi.Truncate(ansi.TrimLeftWidth(rest, pre), r.c1-r.c0))
		if i == cur {
			b.WriteString(hlCur + mid + hlCurOf)
		} else {
			b.WriteString(hlOn + mid + hlOff)
		}
		rest, done = ansi.TrimLeftWidth(rest, r.c1-done), r.c1
	}
	b.WriteString(rest)
	return b.String()
}

// paint returns the text to draw for row i of the current mode (live line when
// not wrapping, visual row when wrapping), before horizontal scrolling, with
// search matches highlighted. vr is m.vrows() when wrapping.
func (m Model) paint(vr []vrow, i int) string {
	var text string
	var idx, c0 int
	if m.wrapOn() {
		r := vr[i]
		text, idx, c0 = r.text, r.line, r.col
	} else {
		idx = m.head + i
		text = m.lines[idx]
	}
	if m.query == "" {
		return text
	}
	ms := m.matchesOf(idx)
	if len(ms) == 0 {
		return text
	}
	w := ansi.Width(text)
	var rs []matchRef
	cur := -1
	for _, s := range ms {
		if s.c1 <= c0 || s.c0 >= c0+w {
			continue
		}
		if m.curSet && s.line == m.cur.line && s.start == m.cur.start {
			cur = len(rs)
		}
		s.c0, s.c1 = max(s.c0, c0)-c0, min(s.c1, c0+w)-c0
		rs = append(rs, s)
	}
	if len(rs) == 0 {
		return text
	}
	return highlight(text, rs, cur)
}

// Search sets the query: every case-insensitive occurrence is highlighted and
// the first match at or below the top visible line becomes the current one and
// is scrolled into view. It returns the number of matches. An empty q clears
// the search. While a query is set, the NextMatch and PrevMatch keys ("n" and
// "N" by default) move between matches.
func (m *Model) Search(q string) int {
	m.query, m.curSet = q, false
	all := m.allMatches()
	if len(all) == 0 {
		return 0
	}
	top := 0 // first visible logical line, as an index into lines
	if m.wrapOn() {
		if vr := m.vrows(); len(vr) > 0 {
			top = vr[min(m.yOffset, len(vr)-1)].line
		}
	} else {
		top = m.head + m.yOffset
	}
	pick := sort.Search(len(all), func(i int) bool { return all[i].line >= top })
	if pick == len(all) {
		pick = 0
	}
	m.goTo(all[pick])
	return len(all)
}

// SearchQuery returns the active query, "" if none.
func (m Model) SearchQuery() string { return m.query }

// MatchCount returns how many matches the active query has in the content.
func (m Model) MatchCount() int { return len(m.allMatches()) }

// CurrentMatch returns the 1-based index of the current match, or 0.
func (m Model) CurrentMatch() int {
	if !m.curSet {
		return 0
	}
	for i, s := range m.allMatches() {
		if s.line == m.cur.line && s.start == m.cur.start {
			return i + 1
		}
	}
	return 0
}

// NextMatch moves to the next match, wrapping from the last to the first, and
// scrolls it into view. It reports whether there was one.
func (m *Model) NextMatch() bool { return m.step(1) }

// PrevMatch moves to the previous match, wrapping from the first to the last.
func (m *Model) PrevMatch() bool { return m.step(-1) }

func (m *Model) step(dir int) bool {
	all := m.allMatches()
	if len(all) == 0 {
		return false
	}
	at := -1
	if m.curSet {
		for i, s := range all {
			if s.line == m.cur.line && s.start == m.cur.start {
				at = i
				break
			}
		}
	}
	var next int
	switch {
	case at < 0 && dir > 0:
		next = 0
	case at < 0:
		next = len(all) - 1
	default:
		next = (at + dir + len(all)) % len(all)
	}
	m.goTo(all[next])
	return true
}

// goTo makes mt the current match and scrolls it into view.
func (m *Model) goTo(mt matchRef) {
	m.cur, m.curSet = mt, true
	ri := mt.line - m.head
	if m.wrapOn() {
		vr := m.vrows()
		first := sort.Search(len(vr), func(i int) bool { return vr[i].line >= mt.line })
		ri = first
		for j := first; j < len(vr) && vr[j].line == mt.line; j++ {
			if vr[j].col <= mt.c0 {
				ri = j
			}
		}
	} else if mt.c1 > m.xOffset+m.Width || mt.c0 < m.xOffset {
		m.setXOffset(max(0, mt.c0-m.Width/2))
	}
	if ri < m.yOffset || ri >= m.yOffset+m.Height {
		m.setYOffset(ri - m.Height/2)
	}
}
