package textinput

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/keymap"
)

func ringInput() Model {
	m := New()
	m.KillRing = true
	m.Focus()
	return m
}

func alt(m Model, r rune) Model {
	m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r, Mod: input.ModAlt})
	return m
}

func home(m Model) Model { return ck2(m, 'a') }

func ck2(m Model, r rune) Model { return ctrl(m, r) }

func TestKillAndYankWord(t *testing.T) {
	m := typed(ringInput(), "one two three")
	m = ctrl(m, 'w')
	if m.Value() != "one two " {
		t.Fatalf("after Ctrl+W: %q", m.Value())
	}
	m = ctrl(m, 'w') // a second kill in a row joins the first
	if m.Value() != "one " {
		t.Fatalf("after the second Ctrl+W: %q", m.Value())
	}
	m = ctrl(m, 'y')
	if m.Value() != "one two three" {
		t.Errorf("Ctrl+Y = %q, want the two kills back in order", m.Value())
	}
}

func TestKillToEndAndStartAndMixedRun(t *testing.T) {
	m := typed(ringInput(), "abc def")
	m = home(m)
	for i := 0; i < 3; i++ {
		m = key(m, tui.KeyRight)
	}
	m = ctrl(m, 'k')
	if m.Value() != "abc" {
		t.Fatalf("Ctrl+K: %q", m.Value())
	}
	m = home(m)
	m = ctrl(m, 'y')
	if m.Value() != " defabc" {
		t.Errorf("yank at the start: %q", m.Value())
	}
	// Ctrl+U kills to the start and Ctrl+Y puts it back.
	m = ctrl(ctrl(typed(ringInput(), "gone"), 'u'), 'y')
	if m.Value() != "gone" {
		t.Errorf("Ctrl+U then Ctrl+Y: %q", m.Value())
	}
	// A backward kill then a forward kill is one entry in document order.
	m = home(typed(ringInput(), "one two three"))
	for i := 0; i < 2; i++ {
		m, _ = m.Update(tui.Key{Type: tui.KeyRight, Mod: input.ModCtrl})
	}
	m = ctrl(m, 'w') // kills "two"
	m = ctrl(m, 'k') // kills " three" right after: the same entry
	if m.Value() != "one " || m.kill.Len() != 1 {
		t.Fatalf("after the two kills: %q, %d entries", m.Value(), m.kill.Len())
	}
	if m = ctrl(m, 'y'); m.Value() != "one two three" {
		t.Errorf("yank of a mixed run: %q", m.Value())
	}
}

func TestYankPopCyclesTheRing(t *testing.T) {
	m := ringInput()
	m = ctrl(typed(m, "first"), 'u')
	m = ctrl(typed(m, "second"), 'u')
	m = ctrl(m, 'y')
	if m.Value() != "second" {
		t.Fatalf("yank: %q", m.Value())
	}
	m = alt(m, 'y')
	if m.Value() != "first" {
		t.Fatalf("yank-pop: %q", m.Value())
	}
	m = alt(m, 'y')
	if m.Value() != "second" {
		t.Fatalf("yank-pop round: %q", m.Value())
	}
	m = typed(m, "x") // any other key ends the yank
	m = alt(m, 'y')
	if m.Value() != "secondx" {
		t.Errorf("Alt+Y after typing changed the text: %q", m.Value())
	}
}

func TestYankIsOneUndoStep(t *testing.T) {
	m := ctrl(typed(ringInput(), "word"), 'u')
	m = typed(m, "keep")
	m = ctrl(m, 'y')
	if m.Value() != "keepword" {
		t.Fatalf("%q", m.Value())
	}
	m = ctrl(m, 'z')
	if m.Value() != "keep" {
		t.Errorf("undo of a yank = %q", m.Value())
	}
}

func TestCtrlYStillRedoesWhenTheRingIsEmpty(t *testing.T) {
	m := typed(ringInput(), "ab")
	m = ctrl(m, 'z')
	if m.Value() == "ab" {
		t.Fatal("setup: undo did nothing")
	}
	m = ctrl(m, 'y')
	if m.Value() != "ab" {
		t.Errorf("Ctrl+Y with an empty ring = %q, want the redo", m.Value())
	}
}

func TestRingOffChangesNothing(t *testing.T) {
	m := New()
	m.Focus()
	m = ctrl(typed(m, "abc"), 'u')
	if m.Value() != "" {
		t.Fatalf("Ctrl+U: %q", m.Value())
	}
	m = ctrl(m, 'z') // undo the deletion
	m = ctrl(m, 'y') // the redo it has always been: no yank of "abc"
	if m.Value() != "" {
		t.Errorf("Ctrl+Y with KillRing off = %q, want the redo", m.Value())
	}
	if bs := m.Bindings(); hasDesc(bs, "yank") {
		t.Error("yank is listed with KillRing off")
	}
	r := ringInput()
	if !hasDesc(r.Bindings(), "yank") || !hasDesc(r.Bindings(), "yank older") {
		t.Error("yank is not listed with KillRing on")
	}
	// With the ring on the same keys yank instead.
	r = ctrl(typed(r, "abc"), 'u')
	r = ctrl(r, 'z')
	if r = ctrl(r, 'y'); r.Value() != "abcabc" {
		t.Errorf("Ctrl+Y with KillRing on = %q, want a yank", r.Value())
	}
}

func hasDesc(bs []keymap.Binding, desc string) bool {
	for _, b := range bs {
		if b.Desc == desc {
			return true
		}
	}
	return false
}

func TestAKillOverASelectionIsNotSaved(t *testing.T) {
	m := typed(ringInput(), "hello")
	m, _ = m.Update(tui.Key{Type: tui.KeyLeft, Mod: input.ModShift})
	m = ctrl(m, 'k')
	if m.Value() != "hell" {
		t.Fatalf("Ctrl+K over a selection: %q", m.Value())
	}
	if m.kill.Len() != 0 {
		t.Error("the selection was saved in the ring")
	}
}

func TestPasswordsNeverReachTheRing(t *testing.T) {
	m := ringInput()
	m.DisableCopy = true
	m = ctrl(typed(m, "secret"), 'u')
	if m.kill.Len() != 0 {
		t.Error("a field with DisableCopy saved a kill")
	}
}

func TestYankHonoursCharLimit(t *testing.T) {
	m := ringInput()
	m.CharLimit = 5
	m = ctrl(typed(m, "abcde"), 'u')
	m = typed(m, "xy")
	m = ctrl(m, 'y')
	if m.Value() != "xyabc" {
		t.Errorf("yank over the limit = %q", m.Value())
	}
}

func TestTwoBackwardKillsOfDifferentKindsJoinInOrder(t *testing.T) {
	m := typed(ringInput(), "one two")
	m = ctrl(m, 'w') // kills "two"
	m = ctrl(m, 'u') // kills "one " before it
	if m.Value() != "" {
		t.Fatalf("setup: %q", m.Value())
	}
	if m = ctrl(m, 'y'); m.Value() != "one two" {
		t.Errorf("Ctrl+W then Ctrl+U yanked %q, want one two", m.Value())
	}
}

func TestYankPopReplacesOnlyTheYankedText(t *testing.T) {
	m := ringInput()
	m = ctrl(typed(m, "first"), 'u')
	m = ctrl(typed(m, "second"), 'u')
	m = typed(m, "pre-")
	m = ctrl(m, 'y')
	if m.Value() != "pre-second" {
		t.Fatalf("yank: %q", m.Value())
	}
	m = alt(m, 'y')
	if m.Value() != "pre-first" {
		t.Errorf("first pop: %q", m.Value())
	}
	m = alt(m, 'y')
	if m.Value() != "pre-second" {
		t.Errorf("second pop: %q", m.Value())
	}
	m = typed(m, "!")
	if m.Value() != "pre-second!" {
		t.Errorf("typing after a pop: %q", m.Value())
	}
}

func TestPopAfterAYankOverASelection(t *testing.T) {
	m := ringInput()
	m = ctrl(typed(m, "A"), 'u')
	m = ctrl(typed(m, "B"), 'u')
	m = typed(m, "xyz")
	m = home(m)
	for i := 0; i < 2; i++ {
		m, _ = m.Update(tui.Key{Type: tui.KeyRight, Mod: input.ModShift})
	}
	if m = ctrl(m, 'y'); m.Value() != "Bz" {
		t.Fatalf("yank over a selection: %q", m.Value())
	}
	if m = alt(m, 'y'); m.Value() != "Az" {
		t.Errorf("pop after it: %q", m.Value())
	}
}
