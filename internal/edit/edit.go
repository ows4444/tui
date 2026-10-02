// Package edit is the grapheme- and width-aware text-editing core shared by
// the input widgets. An Editor holds its text as extended grapheme clusters
// and keeps the cursor as a cluster index, so Left, Right, Backspace and
// Delete always act on whole user-perceived characters, and column math
// goes through ansi.Width.
package edit

import (
	"strings"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
)

// Editor is a single line of text as clusters plus a cursor.
// The zero value is an empty editor.
type Editor struct {
	cl     []string
	w      []int // column width of each cluster, parallel to cl
	cursor int   // cluster index, 0..Len()

	hist    History
	histMax int // undo bound; see SetHistoryLimit
	anchor  int // selection anchor as cluster index + 1; 0 = no selection
}

// Split segments s into grapheme clusters. ansi has no exported segmenter,
// so a rune joins the current cluster when it is zero-width or when the
// pair renders narrower than the sum of its parts (combining marks, ZWJ
// sequences, skin tones, flags, Hangul jamo, conjuncts). Control runes
// always stand alone.
func Split(s string) []string {
	var out []string
	start, curW := 0, 0
	for i, r := range s {
		rw := ansi.Width(string(r))
		if i > 0 && !isCtl(r) && !isCtl(lastRune(s[start:i])) {
			joined := ansi.Width(s[start : i+utf8.RuneLen(r)])
			if rw == 0 || joined < curW+rw {
				curW = joined
				continue
			}
		}
		if i > 0 {
			out = append(out, s[start:i])
		}
		start, curW = i, rw
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func isCtl(r rune) bool { return r < 0x20 || r == 0x7f }

func lastRune(s string) rune { r, _ := utf8.DecodeLastRuneInString(s); return r }

func widths(cl []string) []int {
	w := make([]int, len(cl))
	for i, c := range cl {
		w[i] = ansi.Width(c)
	}
	return w
}

// Len is the number of clusters.
func (e *Editor) Len() int { return len(e.cl) }

// Cursor is the cursor position as a cluster index.
func (e *Editor) Cursor() int { return e.cursor }

// SetCursor moves the cursor to cluster index pos, clamped.
func (e *Editor) SetCursor(pos int) { e.cursor = min(max(pos, 0), len(e.cl)) }

// String returns the text.
func (e *Editor) String() string { return strings.Join(e.cl, "") }

// Cluster returns cluster i.
func (e *Editor) Cluster(i int) string { return e.cl[i] }

// Set replaces the text, keeping at most limit clusters when limit > 0, and
// puts the cursor at the end.
func (e *Editor) Set(s string, limit int) {
	e.cl = Split(s)
	if limit > 0 && len(e.cl) > limit {
		e.cl = e.cl[:limit]
	}
	e.w = widths(e.cl)
	e.cursor = len(e.cl)
	e.hist.Clear()
	e.anchor = 0
}

// Reset clears the text, the history and the selection.
func (e *Editor) Reset() {
	e.cl, e.w, e.cursor, e.anchor = nil, nil, 0, 0
	e.hist.Clear()
}

// Insert inserts s at the cursor in one splice, keeping at most limit
// clusters in total when limit > 0 (extra clusters are dropped). The
// cursor ends after the inserted text. A selection is not touched; use
// ReplaceSelection to type over one.
func (e *Editor) Insert(s string, limit int) { e.splice(e.cursor, e.cursor, s, limit) }

// splice replaces clusters [a, b) with s, in one undo step.
func (e *Editor) splice(a, b int, s string, limit int) {
	if s == "" && a == b {
		return
	}
	// Re-segment together with the cluster before the insertion point so a
	// pasted combining mark or joiner attaches to it.
	lo := a
	if lo > 0 {
		lo--
	}
	prefix := strings.Join(e.cl[lo:a], "")
	ins := Split(prefix + s)
	if limit > 0 {
		room := limit - (len(e.cl) - (b - a) - (a - lo))
		if room < 0 {
			room = 0
		}
		if len(ins) > room {
			ins = ins[:room]
		}
	}
	if e.histMax >= 0 {
		insRunes := 0
		for _, c := range ins {
			insRunes += utf8.RuneCountInString(c)
		}
		insRunes = max(insRunes-utf8.RuneCountInString(prefix), 0)
		removed := ""
		if b > a {
			removed = strings.Join(e.cl[a:b], "")
		}
		if insRunes > 0 || removed != "" {
			e.hist.Record(Step{Pos: e.runeOff(a), Del: insRunes, Ins: removed, Cursor: e.runeOff(e.cursor)},
				a == b && s != " " && s != "\n" && utf8.RuneCountInString(s) == 1, e.histMax)
		}
	}
	insW := widths(ins)
	n := len(e.cl) - (b - lo) + len(ins)
	cl := make([]string, 0, n)
	w := make([]int, 0, n)
	cl = append(append(append(cl, e.cl[:lo]...), ins...), e.cl[b:]...)
	w = append(append(append(w, e.w[:lo]...), insW...), e.w[b:]...)
	e.cl, e.w = cl, w
	e.cursor = lo + len(ins)
	e.anchor = 0
}

// runeOff is the rune offset of cluster index i.
func (e *Editor) runeOff(i int) int {
	n := 0
	for _, c := range e.cl[:i] {
		n += utf8.RuneCountInString(c)
	}
	return n
}

// clusterAt is the index of the cluster holding rune offset r (Len() when r is
// at or past the end).
func (e *Editor) clusterAt(r int) int {
	acc := 0
	for i, c := range e.cl {
		n := utf8.RuneCountInString(c)
		if acc+n > r {
			return i
		}
		acc += n
	}
	return len(e.cl)
}

// Backspace removes the cluster before the cursor.
func (e *Editor) Backspace() {
	if e.DeleteSelection() {
		return
	}
	if e.cursor > 0 {
		e.remove(e.cursor-1, e.cursor)
	}
}

// Delete removes the cluster at the cursor.
func (e *Editor) Delete() {
	if e.DeleteSelection() {
		return
	}
	if e.cursor < len(e.cl) {
		e.remove(e.cursor, e.cursor+1)
	}
}

func (e *Editor) remove(i, j int) {
	if j <= i {
		return
	}
	if e.histMax >= 0 {
		e.hist.Record(Step{Pos: e.runeOff(i), Ins: strings.Join(e.cl[i:j], ""), Cursor: e.runeOff(e.cursor)}, false, e.histMax)
	}
	e.anchor = 0
	// Copy-on-write: copies of an Editor share cl and w, so never shift them
	// in place.
	cl := make([]string, 0, len(e.cl)-(j-i))
	w := make([]int, 0, len(e.cl)-(j-i))
	e.cl = append(append(cl, e.cl[:i]...), e.cl[j:]...)
	e.w = append(append(w, e.w[:i]...), e.w[j:]...)
	if e.cursor > i {
		e.cursor = i
	}
}

// Left and Right move the cursor one cluster.
func (e *Editor) Left() {
	if e.cursor > 0 {
		e.cursor--
	}
}
func (e *Editor) Right() {
	if e.cursor < len(e.cl) {
		e.cursor++
	}
}

// Home and End move to the ends.
func (e *Editor) Home() { e.cursor = 0 }
func (e *Editor) End()  { e.cursor = len(e.cl) }

func (e *Editor) space(i int) bool { return e.cl[i] == " " || e.cl[i] == "\t" }

func (e *Editor) wordStart() int {
	i := e.cursor
	for i > 0 && e.space(i-1) {
		i--
	}
	for i > 0 && !e.space(i-1) {
		i--
	}
	return i
}

// WordLeft moves to the start of the previous word.
func (e *Editor) WordLeft() { e.cursor = e.wordStart() }

// WordRight moves to the end of the next word.
func (e *Editor) WordRight() {
	i := e.cursor
	for i < len(e.cl) && e.space(i) {
		i++
	}
	for i < len(e.cl) && !e.space(i) {
		i++
	}
	e.cursor = i
}

// DeleteWordBack removes the word before the cursor (readline Ctrl+W).
func (e *Editor) DeleteWordBack() {
	if e.DeleteSelection() {
		return
	}
	if i := e.wordStart(); i < e.cursor {
		e.remove(i, e.cursor)
	}
}

// DeleteToStart removes everything before the cursor.
func (e *Editor) DeleteToStart() {
	if !e.DeleteSelection() {
		e.remove(0, e.cursor)
	}
}

// DeleteToEnd removes everything from the cursor on.
func (e *Editor) DeleteToEnd() {
	if !e.DeleteSelection() {
		e.remove(e.cursor, len(e.cl))
	}
}

// Width is the total column width of the text.
func (e *Editor) Width() int {
	t := 0
	for _, x := range e.w {
		t += x
	}
	return t
}

// Window returns the cluster range [start, end) to show in at most cols
// columns so that the cursor cluster (or the end-of-text cell when the
// cursor is at the end, which needs one column) stays visible. The window
// never exceeds cols columns. When everything fits it is the whole text.
func (e *Editor) Window(cols int) (start, end int) {
	n := len(e.cl)
	if cols <= 0 || e.Width()+btoi(e.cursor == n) <= cols {
		return 0, n
	}
	// Reserve the cursor cell, then grow outward: prefer to center.
	cw := 1
	if e.cursor < n {
		cw = e.w[e.cursor]
	}
	if cw > cols {
		cw = cols
	}
	start, end = e.cursor, e.cursor
	if e.cursor < n {
		end = e.cursor + 1
	}
	used := cw
	left, right := 0, 0 // columns spent each side
	for {
		grew := false
		if start > 0 && left <= right && used+e.w[start-1] <= cols {
			start--
			left += e.w[start]
			used += e.w[start]
			grew = true
		} else if end < n && used+e.w[end] <= cols {
			right += e.w[end]
			used += e.w[end]
			end++
			grew = true
		} else if start > 0 && used+e.w[start-1] <= cols {
			start--
			left += e.w[start]
			used += e.w[start]
			grew = true
		}
		if !grew {
			return start, end
		}
	}
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
