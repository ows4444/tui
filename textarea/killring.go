package textarea

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/keymap"
)

// killKey applies the kill ring keys and reports whether it took k. It leaves a
// selection, and Ctrl+Y with an empty ring, to the ordinary keys.
func (m *Model) killKey(k tui.Key, km KeyMap) bool {
	_, _, hasSel := m.selection()
	switch {
	case keymap.Matches(k, km.Yank):
		text, r, ok := m.kill.Yank()
		if !ok {
			return false // an empty ring: Ctrl+Y is Redo, as it always was
		}
		start := m.cursor
		if lo, _, ok := m.selection(); ok {
			start = lo // the yank replaces the selection, so it starts where that did
		}
		m.insertRunes([]rune(text), false)
		m.kill, m.yankLen = r, m.cursor-start
		return true
	case keymap.Matches(k, km.YankPop):
		if text, r, ok := m.kill.Pop(); ok {
			n := min(m.yankLen, m.cursor)
			start := m.cursor - n
			m.replace(start, m.cursor, []rune(text))
			m.cursor = start + len([]rune(text))
			m.kill, m.yankLen = r, m.cursor-start
		}
		return true
	case keyHit(k, km.DeleteToStart) && !hasSel:
		lineStart, _ := m.currentLineBounds()
		text := m.value.str(lineStart, m.cursor)
		m.replace(lineStart, m.cursor, nil)
		m.cursor = lineStart
		m.kill = m.kill.Kill(text, true)
		return true
	case keyHit(k, km.DeleteToEnd) && !hasSel:
		_, lineEnd := m.currentLineBounds()
		text := m.value.str(m.cursor, lineEnd)
		m.replace(m.cursor, lineEnd, nil)
		m.kill = m.kill.Kill(text, false)
		return true
	case keyHit(k, km.DeleteWordBack) && !hasSel:
		i := m.wordStartBeforeCursor()
		text := m.value.str(i, m.cursor)
		m.replace(i, m.cursor, nil)
		m.cursor = i
		m.kill = m.kill.Kill(text, true)
		return true
	}
	return false
}
