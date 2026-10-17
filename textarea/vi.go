package textarea

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/vi"
)

// ViMode is the mode of a Model whose Vi field is set.
type ViMode int

// The modes. ViVisual selects characters and ViVisualLine whole lines.
const (
	ViNormal ViMode = iota
	ViInsert
	ViVisual
	ViVisualLine
)

// ViMode returns the current mode; ViNormal when Vi is not set.
func (m Model) ViMode() ViMode {
	if !m.Vi {
		return ViNormal
	}
	return ViMode(m.vi.Mode())
}

// SetViMode puts the Model in ViNormal or ViInsert (other values are ignored),
// for a form that should start in Insert.
func (m *Model) SetViMode(mode ViMode) {
	switch mode {
	case ViNormal, ViInsert:
		m.vi.SetMode(vi.Mode(mode))
		m.sel = 0
	}
}

// ViPending is the part of a command typed so far, such as "2d", for a status
// line; it is empty when none is pending.
func (m Model) ViPending() string { return m.vi.Pending() }

// viKey gives k to the vi engine. It reports false for a key the Model should
// treat as usual: anything in Insert mode but Esc, and Ctrl or Alt keys.
func (m *Model) viKey(k tui.Key) (bool, tui.Cmd) {
	vk, ok := viKeyOf(k)
	if !ok {
		if m.vi.Mode() == vi.Insert || k.Type == tui.KeyCtrl || k.Type == tui.KeyCtrlC || k.Mod.Ctrl() || k.Mod.Alt() {
			return false, nil
		}
		return true, nil // an unmapped key does nothing in a command mode
	}
	res := m.vi.Handle(viBuf{m}, vk)
	if !res.Handled {
		return false, nil
	}
	if res.Yanked {
		return true, tui.WriteClipboard(res.Yank)
	}
	return true, nil
}

// viKeyOf is k as the engine knows it.
func viKeyOf(k tui.Key) (vi.Key, bool) {
	switch k.Type {
	case tui.KeyEsc:
		return vi.Key{S: vi.Esc}, true
	case tui.KeyEnter:
		return vi.Key{S: vi.Enter}, true
	case tui.KeyBackspace:
		return vi.Key{S: vi.Backspace}, true
	case tui.KeyDelete:
		return vi.Key{S: vi.Delete}, true
	case tui.KeyLeft:
		return vi.Key{S: vi.Left}, true
	case tui.KeyRight:
		return vi.Key{S: vi.Right}, true
	case tui.KeyUp:
		return vi.Key{S: vi.Up}, true
	case tui.KeyDown:
		return vi.Key{S: vi.Down}, true
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

// viBuf lets the engine edit the Model.
type viBuf struct{ m *Model }

func (b viBuf) Len() int      { return b.m.value.len() }
func (b viBuf) At(i int) rune { return b.m.value.at(i) }
func (b viBuf) Cursor() int   { return b.m.cursor }
func (b viBuf) SetCursor(i int) {
	b.m.cursor = clamp(i, 0, b.m.value.len())
}
func (b viBuf) Replace(from, to int, ins []rune) { b.m.replace(from, to, ins) }
func (b viBuf) Select(anchor int, on bool) {
	if on {
		b.m.sel = anchor + 1
	} else {
		b.m.sel = 0
	}
}
func (b viBuf) Begin() { b.m.hist.BeginGroup() }
func (b viBuf) End()   { b.m.hist.EndGroup() }
func (b viBuf) Undo()  { b.m.undo() }
func (b viBuf) Redo()  { b.m.redo() }
