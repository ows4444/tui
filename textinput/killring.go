package textinput

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
)

// killKey applies the kill ring keys and reports whether it took k. It leaves a
// selection, and Ctrl+Y with an empty ring, to the ordinary keys.
func (m *Model) killKey(k tui.Key, km KeyMap) bool {
	hasSel := func() bool { _, _, ok := m.ed.Selection(); return ok }
	switch {
	case keymap.Matches(k, km.Yank):
		text, r, ok := m.kill.Yank()
		if !ok {
			return false // an empty ring: Ctrl+Y is Redo, as it always was
		}
		before := m.ed.Cursor()
		if lo, _, ok := m.ed.Selection(); ok {
			before = lo // the yank replaces the selection, so it starts where that did
		}
		m.ed.ReplaceSelection(text, m.CharLimit)
		m.kill, m.yankLen = r, m.ed.Cursor()-before
		return true
	case keymap.Matches(k, km.YankPop):
		if text, r, ok := m.kill.Pop(); ok {
			n := min(m.yankLen, m.ed.Cursor())
			start := m.ed.Cursor() - n
			m.ed.ReplaceBefore(n, text, m.CharLimit)
			m.kill, m.yankLen = r, m.ed.Cursor()-start
		}
		return true
	case keyHit(k, km.DeleteToStart) && !hasSel():
		m.kill = m.kill.Kill(m.ed.KillToStart(), true)
		return true
	case keyHit(k, km.DeleteToEnd) && !hasSel():
		m.kill = m.kill.Kill(m.ed.KillToEnd(), false)
		return true
	case keyHit(k, km.DeleteWordBack) && !hasSel():
		m.kill = m.kill.Kill(m.ed.KillWordBack(), true)
		return true
	}
	return false
}
