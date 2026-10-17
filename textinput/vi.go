package textinput

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/vi"
)

// ViMode is the mode of a Model whose Vi field is set.
type ViMode int

// The modes. ViVisual selects characters.
const (
	ViNormal ViMode = iota
	ViInsert
	ViVisual
)

// ViMode returns the current mode; ViNormal when Vi is not set.
func (m Model) ViMode() ViMode {
	if !m.Vi {
		return ViNormal
	}
	switch m.vi.Mode() {
	case vi.Insert:
		return ViInsert
	case vi.Visual, vi.VisualLine:
		return ViVisual
	}
	return ViNormal
}

// SetViMode puts the Model in ViNormal or ViInsert (ViVisual is ignored), for a
// field that should start in Insert.
func (m *Model) SetViMode(mode ViMode) {
	switch mode {
	case ViNormal:
		m.vi.SetMode(vi.Normal)
	case ViInsert:
		m.vi.SetMode(vi.Insert)
	default:
		return
	}
	m.ed.ClearSelection()
}

// ViPending is the part of a command typed so far, such as "2d", for a status
// line; it is empty when none is pending.
func (m Model) ViPending() string { return m.vi.Pending() }

// viKey gives k to the vi engine. It reports false for a key the Model should
// treat as usual: anything in Insert mode but Esc, Up, Down, Enter and Tab,
// and Ctrl or Alt keys.
func (m *Model) viKey(k tui.Key) (bool, tui.Cmd) {
	vk, ok := viKeyOf(k)
	if !ok {
		if m.vi.Mode() == vi.Insert || k.Type == tui.KeyCtrl || k.Type == tui.KeyCtrlC || k.Mod.Ctrl() || k.Mod.Alt() ||
			k.Type == tui.KeyUp || k.Type == tui.KeyDown || k.Type == tui.KeyEnter || k.Type == tui.KeyTab {
			return false, nil
		}
		return true, nil // an unmapped key does nothing in a command mode
	}
	res := m.vi.Handle(viBuf{m}, vk)
	if !res.Handled {
		return false, nil
	}
	if res.Yanked && !m.DisableCopy {
		return true, tui.WriteClipboard(res.Yank)
	}
	return true, nil
}

// viKeyOf is k as the engine knows it. Up, Down and Enter are the field's.
func viKeyOf(k tui.Key) (vi.Key, bool) {
	switch k.Type {
	case tui.KeyEsc:
		return vi.Key{S: vi.Esc}, true
	case tui.KeyBackspace:
		return vi.Key{S: vi.Backspace}, true
	case tui.KeyDelete:
		return vi.Key{S: vi.Delete}, true
	case tui.KeyLeft:
		return vi.Key{S: vi.Left}, true
	case tui.KeyRight:
		return vi.Key{S: vi.Right}, true
	case tui.KeyHome:
		return vi.Key{S: vi.Home}, true
	case tui.KeyEnd:
		return vi.Key{S: vi.End}, true
	case tui.KeyCtrl:
		if k.Code == 'r' && !k.Mod.Alt() {
			return vi.Key{S: vi.Redo}, true
		}
	case tui.KeySpace:
		return vi.Key{R: ' '}, !k.Mod.Ctrl() && !k.Mod.Alt()
	case tui.KeyRunes:
		if r := []rune(k.Text); len(r) == 1 && !k.Mod.Ctrl() && !k.Mod.Alt() {
			return vi.Key{R: r[0]}, true
		}
	}
	return vi.Key{}, false
}

// viBuf lets the engine edit the field, one grapheme cluster at a time. A
// newline is never inserted: the field has one line.
type viBuf struct{ m *Model }

func (b viBuf) Len() int { return b.m.ed.Len() }
func (b viBuf) At(i int) rune {
	for _, r := range b.m.ed.Cluster(i) {
		return r
	}
	return ' '
}
func (b viBuf) Cursor() int     { return b.m.ed.Cursor() }
func (b viBuf) SetCursor(i int) { b.m.ed.SetCursor(i) }
func (b viBuf) Replace(from, to int, ins []rune) {
	b.m.ed.SetHistoryLimit(b.m.UndoLimit)
	clean := ins[:0:0]
	for _, r := range ins {
		if r != '\n' && r != '\r' {
			clean = append(clean, r)
		}
	}
	b.m.ed.Replace(from, to, string(clean), b.m.CharLimit)
}
func (b viBuf) Select(anchor int, on bool) { b.m.ed.SelectFrom(anchor, on) }
func (b viBuf) Begin()                     { b.m.ed.BeginGroup() }
func (b viBuf) End()                       { b.m.ed.EndGroup() }
func (b viBuf) Undo()                      { b.m.ed.Undo() }
func (b viBuf) Redo()                      { b.m.ed.Redo() }
