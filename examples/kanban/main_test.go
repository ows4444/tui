package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
)

func send(m model, msg tui.Msg) model {
	next, _ := m.Update(msg)
	return next.(model)
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func shiftKey(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t, Mod: input.ModShift}) }

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready() model { return send(newModel(), tui.ResizeMsg{Width: 100, Height: 30}) }

func text(m model) string { return ansi.StripANSI(m.View()) }

func titles(m model, c int) []string {
	var out []string
	for _, k := range m.b.cols[c] {
		out = append(out, k.title)
	}
	return out
}

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

func mouse(m model, x, y int, b tui.MouseButton, a tui.MouseAction) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Button: b, Action: a})
}

func press(m model, r layout.Rect) model {
	return mouse(m, r.X+1, r.Y, tui.MouseButtonLeft, tui.MouseActionPress)
}

func TestShowsThreeColumnsWithCounts(t *testing.T) {
	out := text(ready())
	for _, want := range []string{"kanban", "To do (4)", "Doing (2)", "Done (2)", "Write the release notes", "Cut rc.1", "+ add", "2 of 8 done"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestKeysMoveTheSelection(t *testing.T) {
	m := runes(ready(), "j")
	if m.col != 0 || m.row != 1 {
		t.Fatalf("j: %d,%d", m.col, m.row)
	}
	m = key(runes(m, "l"), tui.KeyDown)
	if m.col != 1 || m.row != 1 {
		t.Fatalf("l then down: %d,%d", m.col, m.row)
	}
	m = runes(runes(m, "l"), "l")
	if m.col != 2 {
		t.Fatalf("l stops at the last column: %d", m.col)
	}
	m = key(key(m, tui.KeyUp), tui.KeyUp)
	if m.row != 0 {
		t.Fatalf("up stops at the first card: %d", m.row)
	}
}

func TestShiftArrowsMoveTheCardAcross(t *testing.T) {
	m := shiftKey(ready(), tui.KeyRight)
	if got := titles(m, 1); len(got) != 3 || got[2] != "Write the release notes" || m.col != 1 || m.row != 2 {
		t.Fatalf("Doing = %v, selection %d,%d", got, m.col, m.row)
	}
	if len(m.b.cols[0]) != 3 {
		t.Fatalf("To do still has %d", len(m.b.cols[0]))
	}
	m = runes(m, "<")
	if len(m.b.cols[0]) != 4 || m.col != 0 {
		t.Fatalf("< did not move it back: %v", titles(m, 0))
	}
	m = shiftKey(runes(m, "l"), tui.KeyRight)
	m = shiftKey(m, tui.KeyRight)
	if !strings.Contains(m.msg, "No column") {
		t.Fatalf("moving past Done: msg %q", m.msg)
	}
}

func TestReorderWithinAColumn(t *testing.T) {
	m := shiftKey(ready(), tui.KeyDown)
	if got := titles(m, 0); got[0] != "Fix the flaky resize test" || got[1] != "Write the release notes" || m.row != 1 {
		t.Fatalf("%v row %d", got, m.row)
	}
	m = runes(m, "K")
	if titles(m, 0)[0] != "Write the release notes" || m.row != 0 {
		t.Fatalf("K: %v row %d", titles(m, 0), m.row)
	}
	m = runes(m, "K")
	if m.row != 0 {
		t.Fatal("K at the top moved the card")
	}
}

func TestAddRenameDeleteUndo(t *testing.T) {
	m := runes(ready(), "a")
	m = runes(m, "Ship it")
	if !strings.Contains(text(m), "Add to To do: Ship it") {
		t.Fatalf("no editor:\n%s", text(m))
	}
	m = key(m, tui.KeyEnter)
	if got := titles(m, 0); got[len(got)-1] != "Ship it" || m.row != len(got)-1 {
		t.Fatalf("%v", got)
	}
	m = key(runes(m, "e"), tui.KeyBackspace)
	m = key(runes(m, "!"), tui.KeyEnter)
	if titles(m, 0)[4] != "Ship i!" {
		t.Fatalf("rename: %v", titles(m, 0))
	}
	m = runes(m, "d")
	if len(m.b.cols[0]) != 4 || !strings.Contains(m.msg, "u undoes") {
		t.Fatalf("delete: %v %q", titles(m, 0), m.msg)
	}
	m = runes(m, "u")
	if titles(m, 0)[4] != "Ship i!" {
		t.Fatalf("undo: %v", titles(m, 0))
	}
	if m = runes(runes(m, "d"), "u"); len(m.trash) != 0 {
		t.Fatal("trash not emptied")
	}
	if m = runes(m, "u"); !strings.Contains(m.msg, "Nothing to undo") {
		t.Fatalf("msg %q", m.msg)
	}
}

func TestEmptyTitleIsRefusedAndEscCancels(t *testing.T) {
	m := key(runes(ready(), "a"), tui.KeyEnter)
	if m.ed != nil || !strings.Contains(m.msg, "needs a title") || m.b.total() != 8 {
		t.Fatalf("msg %q total %d", m.msg, m.b.total())
	}
	m = key(runes(runes(m, "a"), "x"), tui.KeyEsc)
	if m.ed != nil || m.b.total() != 8 {
		t.Fatal("Esc did not cancel")
	}
}

func TestEmptyColumnStates(t *testing.T) {
	m := ready()
	m.b.cols[1] = nil
	m.col = 1
	m = m.clamp()
	if m.row != -1 || !strings.Contains(text(m), "(empty)") {
		t.Fatalf("row %d:\n%s", m.row, text(m))
	}
	if m = runes(m, "e"); !strings.Contains(m.msg, "Nothing to rename") {
		t.Fatalf("msg %q", m.msg)
	}
	if m = runes(runes(runes(m, "d"), "J"), "j"); m.row != -1 || m.b.total() != 6 {
		t.Fatalf("row %d total %d", m.row, m.b.total())
	}
}

func TestClickSelectsAndDoubleClickRenames(t *testing.T) {
	m := ready()
	m = press(m, rect(t, m, "card-2-1"))
	if m.col != 2 || m.row != 1 {
		t.Fatalf("selection %d,%d", m.col, m.row)
	}
	m = mouse(m, 0, 0, tui.MouseButtonLeft, tui.MouseActionRelease)
	r := rect(t, m, "card-0-2")
	m = send(m, tui.MouseEvent{X: r.X + 1, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: 2})
	if m.ed == nil || string(m.ed.text) != "Reply to the issue about wide characters" {
		t.Fatalf("double-click: %+v", m.ed)
	}
}

func TestDragACardToAnotherColumn(t *testing.T) {
	m := ready()
	from := rect(t, m, "card-0-0")
	to := rect(t, m, "card-1-1")
	m = press(m, from)
	m = mouse(m, to.X+1, to.Y, tui.MouseButtonNone, tui.MouseActionMotion)
	if !strings.Contains(text(m), "Drop on Doing at row 2") {
		t.Fatalf("no drop hint:\n%s", text(m))
	}
	m = mouse(m, to.X+1, to.Y, tui.MouseButtonLeft, tui.MouseActionRelease)
	got := titles(m, 1)
	if len(got) != 3 || got[1] != "Write the release notes" || m.col != 1 || m.row != 1 {
		t.Fatalf("Doing = %v selection %d,%d", got, m.col, m.row)
	}
	if len(m.b.cols[0]) != 3 {
		t.Fatalf("To do = %v", titles(m, 0))
	}
}

func TestDragWithinAColumnAndAClickDoesNotMove(t *testing.T) {
	m := ready()
	m = press(m, rect(t, m, "card-0-0"))
	m = mouse(m, 1, rect(t, m, "card-0-2").Y, tui.MouseButtonNone, tui.MouseActionMotion)
	m = mouse(m, 1, rect(t, m, "card-0-2").Y, tui.MouseButtonLeft, tui.MouseActionRelease)
	if got := titles(m, 0); got[2] != "Write the release notes" {
		t.Fatalf("%v", got)
	}
	before := titles(m, 0)
	m = press(m, rect(t, m, "card-0-1"))
	m = mouse(m, 1, rect(t, m, "card-0-1").Y, tui.MouseButtonLeft, tui.MouseActionRelease)
	if strings.Join(titles(m, 0), "|") != strings.Join(before, "|") {
		t.Fatal("a plain click moved a card")
	}
}

func TestClickAddFootStartsAnEditor(t *testing.T) {
	m := ready()
	m = press(m, rect(t, m, "add-1"))
	if m.ed == nil || m.ed.col != 1 || m.ed.row != -1 {
		t.Fatalf("%+v", m.ed)
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := runes(ready(), "?")
	if !strings.Contains(text(m), "move the card to the next column") {
		t.Fatalf("no help:\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	if _, cmd := ready().Update(tui.Key{Type: tui.KeyCtrlC}); cmd == nil {
		t.Fatal("Ctrl+C did not quit")
	}
	if _, cmd := ready().Update(tui.Key{Type: tui.KeyRunes, Text: "q"}); cmd == nil {
		t.Fatal("q did not quit")
	}
}

func TestLongListScrollsToKeepTheSelection(t *testing.T) {
	m := send(newModel(), tui.ResizeMsg{Width: 60, Height: 10})
	for i := 0; i < 12; i++ {
		m.b, _ = m.b.add(0, "card number "+string(rune('a'+i)))
	}
	for i := 0; i < 15; i++ {
		m = runes(m, "j")
	}
	if !strings.Contains(text(m), "card number l") {
		t.Fatalf("selection scrolled out of view:\n%s", text(m))
	}
}
