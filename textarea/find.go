package textarea

import (
	"unicode"
	"unicode/utf8"
)

// indentBefore is the leading spaces and tabs of the line the cursor is on,
// up to the cursor.
func (m Model) indentBefore() []rune {
	start, _ := m.currentLineBounds()
	var out []rune
	for i := start; i < m.cursor; i++ {
		r := m.value.at(i)
		if r != ' ' && r != '\t' {
			break
		}
		out = append(out, r)
	}
	return out
}

// foldRunes lower-cases s rune by rune, so a match offset in the result is a
// rune offset in s.
func foldRunes(s []rune) []rune {
	out := make([]rune, len(s))
	for i, r := range s {
		out[i] = unicode.ToLower(r)
	}
	return out
}

// matches lists the rune offsets where q occurs in the value, ignoring case,
// without overlap.
func (m Model) matches(q string) []int {
	if q == "" {
		return nil
	}
	pat := foldRunes([]rune(q))
	text := foldRunes([]rune(m.value.str(0, m.value.len())))
	var at []int
	for i := 0; i+len(pat) <= len(text); {
		if runesEqual(text[i:i+len(pat)], pat) {
			at = append(at, i)
			i += len(pat)
		} else {
			i++
		}
	}
	return at
}

func runesEqual(a, b []rune) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Find looks for q, ignoring case, and selects the first occurrence at or after
// the cursor (wrapping to the top), moving the cursor to its end and scrolling
// it into view. It returns how many occurrences there are, and changes nothing
// when there are none or q is empty. FindNext and FindPrev move between them.
func (m *Model) Find(q string) int {
	at := m.matches(q)
	if len(at) == 0 {
		return 0
	}
	n := utf8.RuneCountInString(q)
	lo, _, hasSel := m.selection()
	from := m.cursor
	if hasSel {
		from = lo
	}
	pick := at[0]
	for _, a := range at {
		if a >= from {
			pick = a
			break
		}
	}
	m.selectRange(pick, pick+n)
	return len(at)
}

// FindNext selects the next occurrence of q after the selection (or the
// cursor), wrapping to the first, and reports whether there was one.
func (m *Model) FindNext(q string) bool { return m.step(q, +1) }

// FindPrev selects the occurrence of q before the selection (or the cursor),
// wrapping to the last, and reports whether there was one.
func (m *Model) FindPrev(q string) bool { return m.step(q, -1) }

func (m *Model) step(q string, dir int) bool {
	at := m.matches(q)
	if len(at) == 0 {
		return false
	}
	n := utf8.RuneCountInString(q)
	lo, hi, hasSel := m.selection()
	pick := at[0]
	if dir > 0 {
		from := m.cursor
		if hasSel {
			from = hi
		}
		for _, a := range at {
			if a >= from {
				pick = a
				break
			}
		}
	} else {
		from := m.cursor
		if hasSel {
			from = lo
		}
		pick = at[len(at)-1]
		for i := len(at) - 1; i >= 0; i-- {
			if at[i]+n <= from {
				pick = at[i]
				break
			}
		}
	}
	m.selectRange(pick, pick+n)
	return true
}

// selectRange selects [lo, hi) with the cursor at hi.
func (m *Model) selectRange(lo, hi int) {
	m.sel, m.cursor = lo+1, hi
	m.scrollToCursor()
}

// ReplaceAll replaces every occurrence of q, ignoring case, with repl, as one
// undo step, and returns how many it replaced. The selection is dropped and
// the cursor keeps its place in the text as far as the edits allow. A CharLimit
// is not checked: the text may grow past it.
func (m *Model) ReplaceAll(q, repl string) int {
	at := m.matches(q)
	if len(at) == 0 {
		return 0
	}
	n := utf8.RuneCountInString(q)
	rs := pasteRunes(repl)
	orig := []rune(m.value.str(0, m.value.len()))
	cur0 := m.cursor
	out := make([]rune, 0, len(orig))
	last, shift := 0, 0
	for _, a := range at {
		out = append(out, orig[last:a]...)
		out = append(out, rs...)
		last = a + n
		if a+n <= cur0 {
			shift += len(rs) - n
		}
	}
	out = append(out, orig[last:]...)
	m.replace(0, len(orig), out)
	m.cursor = m.snap(clamp(cur0+shift, 0, m.value.len()))
	m.scrollToCursor()
	return len(at)
}
