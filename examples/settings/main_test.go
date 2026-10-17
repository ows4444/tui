package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func send(m model, msg tui.Msg) model {
	m, _ = update(m, msg)
	return m
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready(w, h int) model { return send(initialModel(), tui.ResizeMsg{Width: w, Height: h}) }

func text(m model) string { return ansi.StripANSI(m.View()) }

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no target %q on screen:\n%s", id, text(m))
	}
	return r
}

func clickAt(m model, x, y int) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

func click(t testing.TB, m model, id string) model {
	r := rect(t, m, id)
	return clickAt(m, r.X+r.W/2, r.Y)
}

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func TestShowsTheFirstCategory(t *testing.T) {
	out := text(ready(80, 24))
	for _, want := range []string{"Settings", "General", "Appearance", "Editor", "Notifications", "Language", "English", "Check for updates", "All changes applied"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestClickCategorySwitchesList(t *testing.T) {
	m := click(t, ready(80, 24), "cat-1")
	if m.cat != 1 || !strings.Contains(text(m), "Font size") {
		t.Fatalf("cat=%d:\n%s", m.cat, text(m))
	}
}

func TestClickSwitchFlipsAndMarksDirty(t *testing.T) {
	m := click(t, ready(80, 24), "ctl-3") // Share usage data, off
	if m.cur["telemetry"] != 1 {
		t.Fatal("clicking the switch did not turn it on")
	}
	out := text(m)
	if !strings.Contains(out, "● 1 unsaved") || !strings.Contains(out, "General ●") {
		t.Fatalf("no dirty marks:\n%s", out)
	}
	m = click(t, m, "ctl-3")
	if len(m.dirtyKeys()) != 0 {
		t.Fatal("flipping back did not clear the mark")
	}
}

func TestChoiceArrowsStepAndWrap(t *testing.T) {
	m := click(t, ready(80, 24), "inc-0")
	if m.cur["lang"] != 1 {
		t.Fatalf("lang = %v after ›, want 1", m.cur["lang"])
	}
	m = click(t, click(t, m, "dec-0"), "dec-0")
	if m.cur["lang"] != 3 {
		t.Fatalf("lang = %v after wrapping back, want 3", m.cur["lang"])
	}
}

func TestSliderClickAndDrag(t *testing.T) {
	m := click(t, ready(80, 24), "cat-1")
	r := rect(t, m, "track-1") // font size, 8 to 32
	m = clickAt(m, r.X+r.W-1, r.Y)
	if m.cur["font"] != 32 || m.drag != 1 {
		t.Fatalf("font=%v drag=%d after clicking the track's end", m.cur["font"], m.drag)
	}
	m = send(m, tui.MouseEvent{X: r.X, Y: r.Y, Action: tui.MouseActionMotion})
	if m.cur["font"] != 8 {
		t.Fatalf("font = %v after dragging to the start, want 8", m.cur["font"])
	}
	m = send(m, tui.MouseEvent{X: r.X + 4, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease})
	if m.drag != -1 {
		t.Fatal("releasing did not end the drag")
	}
	before := m.cur["font"]
	m = send(m, tui.MouseEvent{X: r.X + r.W - 1, Y: r.Y, Action: tui.MouseActionMotion})
	if m.cur["font"] != before {
		t.Fatal("moving the pointer after release still dragged")
	}
}

func TestKeysChangeValues(t *testing.T) {
	m := key(ready(80, 24), tui.KeyRight)
	if m.cur["lang"] != 1 {
		t.Fatalf("Right: lang = %v", m.cur["lang"])
	}
	m = runes(runes(m, "j"), " ") // Check for updates, on -> off
	if m.cur["updates"] != 0 {
		t.Fatal("Space did not flip the switch")
	}
	m = runes(m, "h")
	if m.cur["updates"] != 1 {
		t.Fatal("h did not flip it back")
	}
}

func TestSliderKeysStopAtTheEnds(t *testing.T) {
	m := click(t, click(t, ready(80, 24), "cat-1"), "row-1") // Font size
	for i := 0; i < 40; i++ {
		m = key(m, tui.KeyRight)
	}
	if m.cur["font"] != 32 {
		t.Fatalf("font = %v after many Rights, want 32", m.cur["font"])
	}
	for i := 0; i < 40; i++ {
		m = key(m, tui.KeyLeft)
	}
	if m.cur["font"] != 8 {
		t.Fatalf("font = %v after many Lefts, want 8", m.cur["font"])
	}
}

func TestVolumeNeedsSound(t *testing.T) {
	m := click(t, ready(80, 24), "cat-3")
	m = click(t, m, "ctl-0") // Sound off
	m = click(t, m, "row-1")
	if out := text(m); !strings.Contains(out, "Turn sound on") {
		t.Fatalf("no explanation for the disabled row:\n%s", out)
	}
	r := rect(t, m, "track-1")
	m = clickAt(m, r.X+r.W-1, r.Y)
	if m.cur["volume"] != 60 {
		t.Fatalf("volume changed to %v while sound is off", m.cur["volume"])
	}
	if m.drag != -1 {
		t.Fatal("a disabled slider started a drag")
	}
}

func TestApplyAndReset(t *testing.T) {
	m := click(t, ready(80, 24), "inc-0")
	m = click(t, m, "reset")
	if m.cur["lang"] != 0 || !strings.Contains(text(m), "Changes discarded") {
		t.Fatalf("Reset: lang=%v\n%s", m.cur["lang"], text(m))
	}
	m = click(t, m, "inc-0")
	m = click(t, m, "apply")
	if m.saved["lang"] != 1 || len(m.dirtyKeys()) != 0 {
		t.Fatalf("Apply did not keep the change: saved=%v", m.saved["lang"])
	}
	if !strings.Contains(text(m), "Applied 1 change") {
		t.Fatalf("no confirmation:\n%s", text(m))
	}
}

func TestApplyWithNothingChanged(t *testing.T) {
	m := send(ready(80, 24), tui.Key{Type: tui.KeyCtrl, Code: 's'})
	if !strings.Contains(text(m), "Nothing to apply") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestCtrlShortcuts(t *testing.T) {
	m := send(ready(80, 24), tui.Key{Type: tui.KeyRight})
	m = send(m, tui.Key{Type: tui.KeyCtrl, Code: 'r'})
	if len(m.dirtyKeys()) != 0 {
		t.Fatal("Ctrl+R did not reset")
	}
	m = send(send(m, tui.Key{Type: tui.KeyCtrl, Code: 'n'}), tui.Key{Type: tui.KeyCtrl, Code: 'n'})
	if m.cat != 2 {
		t.Fatalf("cat = %d after Ctrl+N twice, want 2", m.cat)
	}
	if m = key(key(key(key(m, tui.KeyPgDown), tui.KeyPgDown), tui.KeyPgDown), tui.KeyPgDown); m.cat != 2 {
		t.Fatalf("cat = %d after wrapping round, want 2", m.cat)
	}
}

func TestTabMovesBetweenPanes(t *testing.T) {
	m := key(ready(80, 24), tui.KeyTab)
	if m.pane != inCats {
		t.Fatal("Tab did not move to the categories")
	}
	m = key(m, tui.KeyDown)
	if m.cat != 1 {
		t.Fatalf("Down in the category pane: cat = %d", m.cat)
	}
	if m = key(m, tui.KeyEnter); m.pane != inList {
		t.Fatal("Enter did not go into the list")
	}
	if m = key(m, tui.KeyEsc); m.pane != inCats {
		t.Fatal("Esc did not go back to the categories")
	}
}

func TestQuitAsksOnlyWhenChangesWouldBeLost(t *testing.T) {
	if _, cmd := update(ready(80, 24), tui.Key{Type: tui.KeyRunes, Text: "q"}); !isQuit(cmd) {
		t.Fatal("q with nothing changed did not quit")
	}
	m := key(ready(80, 24), tui.KeyRight)
	m, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "q"})
	if isQuit(cmd) || !m.confirm || !strings.Contains(text(m), "Discard 1 unsaved changes?") {
		t.Fatalf("q with changes did not ask:\n%s", text(m))
	}
	if m = runes(m, "n"); m.confirm {
		t.Fatal("n did not cancel")
	}
	m = send(m, tui.Key{Type: tui.KeyRunes, Text: "q"})
	r := rect(t, m, "yes")
	if _, cmd := update(m, tui.MouseEvent{X: r.X + 1, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}); !isQuit(cmd) {
		t.Fatal("the yes button did not quit")
	}
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "y"}); !isQuit(cmd) {
		t.Fatal("y did not quit")
	}
	if _, cmd := update(ready(80, 24).withDirty(), tui.Key{Type: tui.KeyCtrlC}); !isQuit(cmd) {
		t.Fatal("Ctrl+C did not quit at once")
	}
}

func (m model) withDirty() model { return m.set(0, 1) }

func TestHelp(t *testing.T) {
	m := key(ready(80, 24), tui.KeyF1)
	if !strings.Contains(text(m), "apply the changes") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
}

func TestNarrowTerminalFoldsTheCategoryList(t *testing.T) {
	m := ready(40, 12)
	if strings.Contains(text(m), "Notifications") {
		t.Fatalf("the category list shows at 40 columns:\n%s", text(m))
	}
	m = click(t, m, "cat-next")
	if m.cat != 1 || !strings.Contains(text(m), "Appearance (2/4)") {
		t.Fatalf("cat=%d\n%s", m.cat, text(m))
	}
	m = click(t, m, "cat-prev")
	if m.cat != 0 {
		t.Fatalf("cat = %d after ‹", m.cat)
	}
}

func TestHoverIsTracked(t *testing.T) {
	m := ready(80, 24)
	r := rect(t, m, "cat-2")
	m = send(m, tui.MouseEvent{X: r.X + 2, Y: r.Y, Action: tui.MouseActionMotion})
	if m.hover != "cat-2" {
		t.Fatalf("hover = %q", m.hover)
	}
}

func TestWheelMovesTheSelection(t *testing.T) {
	m := send(ready(80, 24), tui.MouseEvent{Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.row != 1 {
		t.Fatalf("row = %d after the wheel, want 1", m.row)
	}
	m = send(send(m, tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress}), tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.row != 0 {
		t.Fatalf("row = %d, want 0", m.row)
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m := ready(80, 24).withDirty()
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}
