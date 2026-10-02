package commandpalette

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func typed(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
	}
	return m
}

func sampleCmds() []Command {
	return []Command{{Name: "alpha"}, {Name: "alpine"}, {Name: "alps"}, {Name: "alpaca"}, {Name: "also"}}
}

func TestKeyMap_RebindDownToCtrlN(t *testing.T) {
	m := typed(New(sampleCmds()...), "al")
	m.KeyMap.Down.Keys = []string{"ctrl+n"}
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	if m.highlight != 0 {
		t.Fatalf("arrow moved after rebind: %d", m.highlight)
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyCtrl, Code: 'n'})
	if m.highlight != 1 {
		t.Fatalf("highlight=%d, want 1", m.highlight)
	}
}

func TestKeyMap_RebindToTypeableKeyStillReachesInputWhenClosed(t *testing.T) {
	m := New(sampleCmds()...)
	m.KeyMap.Down.Keys = []string{"j"}
	m = typed(m, "j")
	if m.Input.Value() != "j" {
		t.Fatalf("query=%q, want j (dropdown closed)", m.Input.Value())
	}
}

func TestKeyMap_ZeroLiteralAndBindings(t *testing.T) {
	m := Model{Commands: sampleCmds(), Input: New().Input}
	m = typed(m, "al")
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	if m.highlight != 1 {
		t.Fatalf("highlight=%d, want 1", m.highlight)
	}
	for _, b := range m.Bindings() {
		if len(b.Keys) == 0 || b.Desc == "" {
			t.Fatalf("incomplete binding %+v", b)
		}
	}
}

func TestMouse_ClickSelectsRowAndWheelMovesHighlight(t *testing.T) {
	m := typed(New(sampleCmds()...), "al")
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 30, H: 6}
	click := tui.MouseEvent{X: 2, Y: 3, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
	if _, cmd := m.Update(click); cmd != nil {
		t.Fatal("mouse off must ignore events")
	}
	m.Mouse = true
	wheel := tui.MouseEvent{X: 2, Y: 1, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress}
	w, _ := m.Update(wheel)
	if w.highlight != 3 {
		t.Fatalf("highlight=%d, want default step 3", w.highlight)
	}
	m.WheelStep = 1
	w, _ = m.Update(wheel)
	if w.highlight != 1 {
		t.Fatalf("highlight=%d, want 1", w.highlight)
	}
	_, cmd := m.Update(click) // screen row 3 is dropdown row 2
	if cmd == nil {
		t.Fatal("click: want SelectedMsg Cmd")
	}
	if got := cmd().(SelectedMsg).Command.Name; got != m.filtered()[2].Name {
		t.Fatalf("selected %q", got)
	}
}
