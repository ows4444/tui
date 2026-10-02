package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func press(m tui.Model, k tui.Key) (model, tui.Cmd) {
	next, cmd := m.Update(k)
	return next.(model), cmd
}

func TestArrowKeysChangeTheCount(t *testing.T) {
	var m tui.Model = model{}
	for _, tc := range []struct {
		key  tui.KeyType
		want int
	}{
		{tui.KeyUp, 1}, {tui.KeyRight, 2}, {tui.KeyDown, 1}, {tui.KeyLeft, 0}, {tui.KeyLeft, -1},
	} {
		next, cmd := press(m, tui.Key{Type: tc.key})
		if cmd != nil || next.count != tc.want {
			t.Fatalf("after key %v: count=%d cmd=%v, want count %d and no Cmd", tc.key, next.count, cmd != nil, tc.want)
		}
		m = next
	}
}

func TestQuitKeys(t *testing.T) {
	for name, k := range map[string]tui.Key{
		"ctrl+c": {Type: tui.KeyCtrlC},
		"q":      {Type: tui.KeyRunes, Text: "q"},
	} {
		if _, cmd := press(model{}, k); !isQuit(cmd) {
			t.Errorf("%s did not quit", name)
		}
	}
	if _, cmd := press(model{}, tui.Key{Type: tui.KeyRunes, Text: "x"}); cmd != nil {
		t.Error("an unbound rune returned a Cmd")
	}
	if _, cmd := press(model{}, tui.Key{Type: tui.KeyRunes}); cmd != nil {
		t.Error("an empty rune key returned a Cmd")
	}
}

func TestViewShowsCountAndSize(t *testing.T) {
	next, _ := model{count: 7}.Update(tui.ResizeMsg{Width: 80, Height: 24})
	out := ansi.StripANSI(next.View())
	for _, want := range []string{"Counter", "Count: 7", "(80x24)"} {
		if !strings.Contains(out, want) {
			t.Errorf("View() missing %q:\n%s", want, out)
		}
	}
}
