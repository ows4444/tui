package textarea

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/keymap"
)

func ringArea() Model {
	m := New()
	m.KillRing = true
	m.Focus()
	return m
}

func kSend(m Model, k tui.Key) Model {
	m, _ = m.Update(k)
	return m
}

func kCtl(m Model, r rune) Model { return kSend(m, tui.Key{Type: tui.KeyCtrl, Code: r}) }

func kAlt(m Model, r rune) Model {
	return kSend(m, tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r, Mod: input.ModAlt})
}

func kType(m Model, s string) Model {
	for _, r := range s {
		if r == '\n' {
			m = kSend(m, tui.Key{Type: tui.KeyEnter})
			continue
		}
		m = kSend(m, rk(r))
	}
	return m
}

func hasDesc(bs []keymap.Binding, desc string) bool {
	for _, b := range bs {
		if b.Desc == desc {
			return true
		}
	}
	return false
}

func TestKillWordsAndYankBack(t *testing.T) {
	m := kType(ringArea(), "one two three")
	m = kCtl(m, 'w')
	m = kCtl(m, 'w')
	if m.Value() != "one " {
		t.Fatalf("after two Ctrl+W: %q", m.Value())
	}
	if m = kCtl(m, 'y'); m.Value() != "one two three" {
		t.Errorf("Ctrl+Y = %q, want the two kills back in order", m.Value())
	}
}

func TestKillsStayOnTheirLine(t *testing.T) {
	m := kType(ringArea(), "first line\nsecond line")
	m = kCtl(m, 'u') // kills "second line" only
	if m.Value() != "first line\n" {
		t.Fatalf("Ctrl+U: %q", m.Value())
	}
	m = kSend(m, tui.Key{Type: tui.KeyUp})
	m = kCtl(m, 'e') // end of the first line
	m = kCtl(m, 'y')
	if m.Value() != "first linesecond line\n" {
		t.Errorf("yank into another line: %q", m.Value())
	}
	m = kCtl(kCtl(kType(ringArea(), "keep\nabc def"), 'a'), 'k')
	if m.Value() != "keep\n" {
		t.Errorf("Ctrl+K from the line start: %q", m.Value())
	}
}

func TestKillToEndAndMixedRun(t *testing.T) {
	m := kCtl(kType(ringArea(), "one two three"), 'a')
	for i := 0; i < 2; i++ {
		m = kSend(m, tui.Key{Type: tui.KeyRight, Mod: input.ModCtrl})
	}
	m = kCtl(m, 'w') // kills "two"
	m = kCtl(m, 'k') // kills " three" right after: the same entry
	if m.Value() != "one " || m.kill.Len() != 1 {
		t.Fatalf("after the two kills: %q, %d entries", m.Value(), m.kill.Len())
	}
	if m = kCtl(m, 'y'); m.Value() != "one two three" {
		t.Errorf("yank of a mixed run: %q", m.Value())
	}
	m = kType(ringArea(), "one two")
	m = kCtl(kCtl(m, 'w'), 'u')
	if m = kCtl(m, 'y'); m.Value() != "one two" {
		t.Errorf("Ctrl+W then Ctrl+U yanked %q", m.Value())
	}
}

func TestYankPopReplacesOnlyTheYankedText(t *testing.T) {
	m := ringArea()
	m = kCtl(kType(m, "first"), 'u')
	m = kCtl(kType(m, "second"), 'u')
	m = kType(m, "pre-")
	if m = kCtl(m, 'y'); m.Value() != "pre-second" {
		t.Fatalf("yank: %q", m.Value())
	}
	if m = kAlt(m, 'y'); m.Value() != "pre-first" {
		t.Errorf("first pop: %q", m.Value())
	}
	if m = kAlt(m, 'y'); m.Value() != "pre-second" {
		t.Errorf("second pop: %q", m.Value())
	}
	if m = kType(m, "!"); m.Value() != "pre-second!" {
		t.Errorf("typing after a pop: %q", m.Value())
	}
	if m = kAlt(m, 'y'); m.Value() != "pre-second!" {
		t.Errorf("Alt+Y after typing changed the text: %q", m.Value())
	}
}

func TestYankIsOneUndoStep(t *testing.T) {
	m := kCtl(kType(ringArea(), "word"), 'u')
	m = kType(m, "keep")
	if m = kCtl(m, 'y'); m.Value() != "keepword" {
		t.Fatalf("%q", m.Value())
	}
	if m = kCtl(m, 'z'); m.Value() != "keep" {
		t.Errorf("undo of a yank = %q", m.Value())
	}
}

func TestCtrlYStillRedoesWhenTheRingIsEmpty(t *testing.T) {
	m := kType(ringArea(), "ab")
	m = kCtl(m, 'z')
	if m.Value() == "ab" {
		t.Fatal("setup: undo did nothing")
	}
	if m = kCtl(m, 'y'); m.Value() != "ab" {
		t.Errorf("Ctrl+Y with an empty ring = %q, want the redo", m.Value())
	}
}

func TestRingOffChangesNothing(t *testing.T) {
	m := New()
	m.Focus()
	m = kCtl(kType(m, "abc"), 'u')
	m = kCtl(m, 'z') // undo the deletion
	if m = kCtl(m, 'y'); m.Value() != "" {
		t.Errorf("Ctrl+Y with KillRing off = %q, want the redo", m.Value())
	}
	if hasDesc(m.Bindings(), "yank") {
		t.Error("yank is listed with KillRing off")
	}
	r := ringArea()
	if !hasDesc(r.Bindings(), "yank") || !hasDesc(r.Bindings(), "yank older") {
		t.Error("yank is not listed with KillRing on")
	}
	r = kCtl(kType(r, "abc"), 'u')
	r = kCtl(r, 'z')
	if r = kCtl(r, 'y'); r.Value() != "abcabc" {
		t.Errorf("Ctrl+Y with KillRing on = %q, want a yank", r.Value())
	}
}

func TestAKillOverASelectionIsNotSaved(t *testing.T) {
	m := kType(ringArea(), "hello")
	m = kSend(m, tui.Key{Type: tui.KeyLeft, Mod: input.ModShift})
	m = kCtl(m, 'k')
	if m.Value() != "hell" {
		t.Fatalf("Ctrl+K over a selection: %q", m.Value())
	}
	if m.kill.Len() != 0 {
		t.Error("the selection was saved in the ring")
	}
}

func TestYankReplacesASelection(t *testing.T) {
	m := kCtl(kType(ringArea(), "xx"), 'u')
	m = kType(m, "abc")
	m = kSend(m, tui.Key{Type: tui.KeyLeft, Mod: input.ModShift})
	m = kCtl(m, 'y')
	if m.Value() != "abxx" {
		t.Errorf("yank over a selection: %q", m.Value())
	}
	if m = kAlt(m, 'y'); m.Value() != "abxx" {
		t.Errorf("a pop with one entry changed the text: %q", m.Value())
	}
}

func TestACustomKeyMapGetsTheYankKeys(t *testing.T) {
	m := ringArea()
	km := DefaultKeyMap()
	km.Yank, km.YankPop = keymap.Binding{}, keymap.Binding{}
	m.KeyMap = km
	m = kCtl(kType(m, "abc"), 'u')
	if m = kCtl(m, 'y'); m.Value() != "abc" {
		t.Errorf("a KeyMap without Yank did not get Ctrl+Y: %q", m.Value())
	}
}

func TestAKillTakesOnlyItsOwnLine(t *testing.T) {
	m := kType(ringArea(), "abc def\nnext line")
	m = kSend(m, tui.Key{Type: tui.KeyUp})
	m = kCtl(m, 'a')
	for i := 0; i < 3; i++ {
		m = kSend(m, tui.Key{Type: tui.KeyRight})
	}
	if m = kCtl(m, 'k'); m.Value() != "abc\nnext line" {
		t.Fatalf("Ctrl+K: %q", m.Value())
	}
	if m = kCtl(m, 'y'); m.Value() != "abc def\nnext line" {
		t.Errorf("the yank held more than the line: %q", m.Value())
	}
}

func TestPopAfterAYankOverASelection(t *testing.T) {
	m := ringArea()
	m = kCtl(kType(m, "A"), 'u')
	m = kCtl(kType(m, "B"), 'u')
	m = kType(m, "xyz")
	m = kCtl(m, 'a')
	for i := 0; i < 2; i++ {
		m = kSend(m, tui.Key{Type: tui.KeyRight, Mod: input.ModShift})
	}
	if m = kCtl(m, 'y'); m.Value() != "Bz" {
		t.Fatalf("yank over a selection: %q", m.Value())
	}
	if m = kAlt(m, 'y'); m.Value() != "Az" {
		t.Errorf("pop after it: %q", m.Value())
	}
}
