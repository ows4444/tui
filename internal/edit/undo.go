package edit

import (
	"strings"
	"unicode/utf8"
)

// SetHistoryLimit sets how many undo steps the Editor keeps: 0 means
// DefaultHistory, a negative n turns history off (and drops what it holds).
// Lowering the limit takes effect as new steps are recorded.
func (e *Editor) SetHistoryLimit(n int) {
	e.histMax = n
	if n < 0 {
		e.hist.Clear()
	}
}

// CanUndo reports whether Undo would change the text.
func (e *Editor) CanUndo() bool { return e.hist.CanUndo() }

// CanRedo reports whether Redo would change the text.
func (e *Editor) CanRedo() bool { return e.hist.CanRedo() }

// HistoryLen is the number of undo steps currently available.
func (e *Editor) HistoryLen() int { return e.hist.Len() }

// Undo reverts the newest edit, restoring the text and the cursor as they were
// before it, and reports whether there was one. Set and Reset forget history.
func (e *Editor) Undo() bool {
	s, ok := e.hist.PopUndo()
	if !ok {
		return false
	}
	e.hist.PushRedo(e.applyStep(s))
	return true
}

// Redo re-applies the edit Undo reverted, and reports whether there was one.
func (e *Editor) Redo() bool {
	s, ok := e.hist.PopRedo()
	if !ok {
		return false
	}
	e.hist.PushUndo(e.applyStep(s), e.histMax)
	return true
}

// applyStep applies s and returns the step that reverses it.
func (e *Editor) applyStep(s Step) Step {
	rs := []rune(e.String())
	pos := min(max(s.Pos, 0), len(rs))
	end := min(pos+s.Del, len(rs))
	inv := Step{Pos: pos, Del: utf8.RuneCountInString(s.Ins), Ins: string(rs[pos:end]), Cursor: e.runeOff(e.cursor)}
	out := make([]rune, 0, len(rs)-(end-pos)+inv.Del)
	out = append(append(append(out, rs[:pos]...), []rune(s.Ins)...), rs[end:]...)
	e.cl = Split(string(out))
	e.w = widths(e.cl)
	e.cursor = e.clusterAt(s.Cursor)
	e.anchor = 0
	return inv
}

// SetAnchor starts a selection at the cursor unless one is active, so that
// moving the cursor afterwards extends it.
func (e *Editor) SetAnchor() {
	if e.anchor == 0 {
		e.anchor = e.cursor + 1
	}
}

// ClearSelection drops the selection without touching the text.
func (e *Editor) ClearSelection() { e.anchor = 0 }

// SelectAll selects the whole text and puts the cursor at its end.
func (e *Editor) SelectAll() {
	e.anchor, e.cursor = 1, len(e.cl)
}

// Selection returns the selected cluster range [lo, hi); ok is false when
// nothing is selected.
func (e *Editor) Selection() (lo, hi int, ok bool) {
	if e.anchor == 0 {
		return 0, 0, false
	}
	a := min(e.anchor-1, len(e.cl))
	lo, hi = min(a, e.cursor), max(a, e.cursor)
	return lo, hi, lo != hi
}

// SelectedText returns the selected text, or "" when nothing is selected.
func (e *Editor) SelectedText() string {
	lo, hi, ok := e.Selection()
	if !ok {
		return ""
	}
	return strings.Join(e.cl[lo:hi], "")
}

// DeleteSelection removes the selected text as one undo step and reports
// whether there was a selection.
func (e *Editor) DeleteSelection() bool {
	lo, hi, ok := e.Selection()
	e.anchor = 0
	if !ok {
		return false
	}
	e.remove(lo, hi)
	e.cursor = lo
	return true
}

// ReplaceSelection inserts s in place of the selection (at the cursor when
// there is none) as one undo step, keeping at most limit clusters when
// limit > 0.
func (e *Editor) ReplaceSelection(s string, limit int) {
	lo, hi, ok := e.Selection()
	if !ok {
		e.anchor = 0
		e.Insert(s, limit)
		return
	}
	if s == "" {
		e.DeleteSelection()
		return
	}
	e.splice(lo, hi, s, limit)
}
