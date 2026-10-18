package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
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

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func ready() model { return send(newModel(), tui.ResizeMsg{Width: 100, Height: 30}) }

func text(m model) string { return ansi.StripANSI(m.View()) }

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

// px is the screen position of the middle of canvas pixel (x, y).
func px(t testing.TB, m model, x, y int) (int, int) {
	r := rect(t, m, "canvas")
	return r.X + 2*x, r.Y + y
}

func mouse(m model, x, y int, b tui.MouseButton, a tui.MouseAction) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Button: b, Action: a})
}

// drag presses at one pixel, moves through the second and releases.
func drag(t testing.TB, m model, b tui.MouseButton, from, to pt) model {
	x, y := px(t, m, from.x, from.y)
	m = mouse(m, x, y, b, tui.MouseActionPress)
	x, y = px(t, m, to.x, to.y)
	m = mouse(m, x, y, b, tui.MouseActionMotion)
	return mouse(m, x, y, b, tui.MouseActionRelease)
}

func count(g grid, c uint8) int {
	n := 0
	for _, v := range g.px {
		if v == c {
			n++
		}
	}
	return n
}

func TestFirstScreen(t *testing.T) {
	out := text(ready())
	for _, want := range []string{"paint", "▸Pen", "Eraser", "Line", "Rect", "Fill", "brush 1", "[1██]", " 8██", "┌", "└", "49x24 pixels", "saved"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestCanvasFollowsTheWindowAndKeepsTheDrawing(t *testing.T) {
	m := drag(t, ready(), tui.MouseButtonLeft, pt{2, 2}, pt{2, 2})
	m = send(m, tui.ResizeMsg{Width: 60, Height: 20})
	if m.g.w != 29 || m.g.h != 14 || m.g.at(pt{2, 2}) != 1 {
		t.Fatalf("%dx%d, pixel %d", m.g.w, m.g.h, m.g.at(pt{2, 2}))
	}
	m = send(m, tui.ResizeMsg{Width: 40, Height: 10})
	if m.g.w != 19 || m.g.h != 4 || !strings.Contains(text(m), "19x4 pixels") {
		t.Fatalf("%dx%d", m.g.w, m.g.h)
	}
}

func TestLeftDragDrawsAConnectedStroke(t *testing.T) {
	m := drag(t, ready(), tui.MouseButtonLeft, pt{3, 3}, pt{12, 8})
	if count(m.g, 1) < 10 || m.g.at(pt{3, 3}) != 1 || m.g.at(pt{12, 8}) != 1 || !m.dirty {
		t.Fatalf("%d pixels", count(m.g, 1))
	}
	if !strings.Contains(text(m), "██") || !strings.Contains(text(m), "not saved") {
		t.Fatalf("not drawn:\n%s", text(m))
	}
}

func TestDragThroughSeveralMovesJoinsThem(t *testing.T) {
	m := ready()
	x, y := px(t, m, 1, 1)
	m = mouse(m, x, y, tui.MouseButtonLeft, tui.MouseActionPress)
	for _, p := range []pt{{5, 1}, {5, 6}} {
		x, y = px(t, m, p.x, p.y)
		m = mouse(m, x, y, tui.MouseButtonLeft, tui.MouseActionMotion)
	}
	m = mouse(m, x, y, tui.MouseButtonLeft, tui.MouseActionRelease)
	if m.g.at(pt{3, 1}) != 1 || m.g.at(pt{5, 4}) != 1 || len(m.undo) != 1 {
		t.Fatalf("pixels %d, undo steps %d", count(m.g, 1), len(m.undo))
	}
}

func TestRightDragErases(t *testing.T) {
	m := drag(t, ready(), tui.MouseButtonLeft, pt{1, 1}, pt{10, 1})
	m = drag(t, m, tui.MouseButtonRight, pt{4, 1}, pt{6, 1})
	if m.g.at(pt{5, 1}) != 0 || m.g.at(pt{2, 1}) != 1 || m.g.at(pt{8, 1}) != 1 {
		t.Fatalf("\n%s", m.g.text())
	}
}

func TestToolAndColourClicks(t *testing.T) {
	m := ready()
	r := rect(t, m, "tool-2")
	m = mouse(m, r.X+1, r.Y, tui.MouseButtonLeft, tui.MouseActionPress)
	if m.tool != line || !strings.Contains(text(m), "▸Line") {
		t.Fatalf("tool %v", m.tool)
	}
	r = rect(t, m, "col-5")
	m = mouse(m, r.X+1, r.Y, tui.MouseButtonLeft, tui.MouseActionPress)
	if m.colour != 5 || !strings.Contains(text(m), "[5██]") {
		t.Fatalf("colour %d", m.colour)
	}
	m = drag(t, m, tui.MouseButtonLeft, pt{2, 2}, pt{9, 2})
	if m.g.at(pt{5, 2}) != 5 {
		t.Fatalf("drew in the wrong colour: %d", m.g.at(pt{5, 2}))
	}
}

func TestLineAndRectShowAPreviewThenCommit(t *testing.T) {
	m := runes(ready(), "r")
	x, y := px(t, m, 2, 2)
	m = mouse(m, x, y, tui.MouseButtonLeft, tui.MouseActionPress)
	x, y = px(t, m, 8, 6)
	m = mouse(m, x, y, tui.MouseButtonLeft, tui.MouseActionMotion)
	if !m.g.empty() || !strings.Contains(text(m), "other corner") || !strings.Contains(text(m), "██") {
		t.Fatalf("the preview should show but not be saved yet:\n%s", text(m))
	}
	m = mouse(m, x, y, tui.MouseButtonLeft, tui.MouseActionRelease)
	if m.g.at(pt{2, 2}) != 1 || m.g.at(pt{8, 6}) != 1 || m.g.at(pt{5, 4}) != 0 || len(m.undo) != 1 {
		t.Fatalf("\n%s", m.g.text())
	}
}

func TestFillClickAndEraserTool(t *testing.T) {
	m := drag(t, runes(ready(), "r"), tui.MouseButtonLeft, pt{2, 2}, pt{8, 6})
	m = runes(runes(m, "f"), "3")
	x, y := px(t, m, 5, 4)
	m = mouse(m, x, y, tui.MouseButtonLeft, tui.MouseActionPress)
	m = mouse(m, x, y, tui.MouseButtonLeft, tui.MouseActionRelease)
	if m.g.at(pt{5, 4}) != 3 || m.g.at(pt{2, 2}) != 1 || m.g.at(pt{0, 0}) != 0 {
		t.Fatalf("\n%s", m.g.text())
	}
	m = drag(t, runes(m, "e"), tui.MouseButtonLeft, pt{5, 4}, pt{5, 4})
	if m.g.at(pt{5, 4}) != 0 {
		t.Fatal("the eraser tool did not erase")
	}
}

func TestUndoRedoAndClear(t *testing.T) {
	m := drag(t, ready(), tui.MouseButtonLeft, pt{1, 1}, pt{5, 1})
	m = drag(t, m, tui.MouseButtonLeft, pt{1, 3}, pt{5, 3})
	m = runes(m, "u")
	if m.g.at(pt{3, 3}) != 0 || m.g.at(pt{3, 1}) != 1 {
		t.Fatalf("undo wrong:\n%s", m.g.text())
	}
	m = runes(m, "U")
	if m.g.at(pt{3, 3}) != 1 {
		t.Fatal("redo did not bring it back")
	}
	m = runes(m, "c")
	if !m.g.empty() || !strings.Contains(m.msg, "u undoes") {
		t.Fatal("clear failed")
	}
	if m = runes(runes(m, "u"), "u"); m.g.at(pt{3, 1}) != 1 || m.g.at(pt{3, 3}) != 0 {
		t.Fatalf("undoing the clear and a stroke:\n%s", m.g.text())
	}
	m = runes(runes(m, "u"), "u")
	if !strings.Contains(m.msg, "Nothing to undo") {
		t.Fatalf("msg %q", m.msg)
	}
}

func TestANewStrokeDropsTheRedoHistory(t *testing.T) {
	m := drag(t, ready(), tui.MouseButtonLeft, pt{1, 1}, pt{5, 1})
	m = runes(m, "u")
	m = drag(t, m, tui.MouseButtonLeft, pt{1, 2}, pt{5, 2})
	if m = runes(m, "U"); !strings.Contains(m.msg, "Nothing to redo") {
		t.Fatalf("msg %q", m.msg)
	}
}

func TestKeyboardDrawsWithSpace(t *testing.T) {
	m := ready()
	m = runes(m, " ")
	for i := 0; i < 4; i++ {
		m = key(m, tui.KeyRight)
	}
	m = runes(m, " ")
	if count(m.g, 1) != 5 || m.last != nil {
		t.Fatalf("pen stroke with the keys: %d pixels, last %v", count(m.g, 1), m.last)
	}
	m = send(m, tui.Key{Type: tui.KeyDown, Mod: input.ModShift})
	if m.cur.y != 5 {
		t.Fatalf("Shift+Down moved to %d", m.cur.y)
	}
	m = runes(runes(m, "l"), " ")
	m = key(runes(m, ""), tui.KeyRight)
	m = key(m, tui.KeyRight)
	m = runes(m, " ")
	if m.g.at(pt{6, 5}) != 1 || m.g.at(pt{5, 5}) != 1 {
		t.Fatalf("keyboard line:\n%s", m.g.text())
	}
}

func TestEscCancelsAShape(t *testing.T) {
	m := runes(runes(ready(), "r"), " ")
	m = key(m, tui.KeyRight)
	if m.anchor == nil {
		t.Fatal("no shape started")
	}
	if m = key(m, tui.KeyEsc); m.anchor != nil || !m.g.empty() {
		t.Fatal("Esc did not cancel")
	}
}

func TestBrushSizeKeysAndWheel(t *testing.T) {
	m := runes(runes(ready(), "]"), "]")
	if m.brush != 3 || !strings.Contains(text(m), "brush 3") {
		t.Fatalf("brush %d", m.brush)
	}
	m = runes(m, "]")
	if m.brush != 3 {
		t.Fatal("brush past 3")
	}
	x, y := px(t, m, 4, 4)
	m = mouse(m, x, y, tui.MouseButtonWheelDown, tui.MouseActionPress)
	if m.brush != 2 {
		t.Fatalf("wheel: brush %d", m.brush)
	}
	m = drag(t, m, tui.MouseButtonLeft, pt{5, 5}, pt{5, 5})
	if count(m.g, 1) != 4 {
		t.Fatalf("a size-2 dab is %d pixels", count(m.g, 1))
	}
}

func TestSaveWritesTheTextPicture(t *testing.T) {
	m := drag(t, ready(), tui.MouseButtonLeft, pt{0, 0}, pt{3, 0})
	m.path = filepath.Join(t.TempDir(), "p.txt")
	m = runes(m, "s")
	got, err := os.ReadFile(m.path)
	if err != nil || string(got) != "....\n" {
		t.Fatalf("%q %v", got, err)
	}
	if m.dirty || !strings.Contains(text(m), "Saved") {
		t.Fatal("still marked unsaved")
	}
	m.path = filepath.Join(t.TempDir(), "missing", "p.txt")
	if m = runes(m, "s"); !strings.Contains(m.msg, "Could not save") {
		t.Fatalf("msg %q", m.msg)
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := runes(ready(), "?")
	if !strings.Contains(text(m), "start and end a stroke or shape") {
		t.Fatalf("no help:\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	if _, c := ready().Update(tui.Key{Type: tui.KeyCtrlC}); c == nil {
		t.Fatal("Ctrl+C did not quit")
	}
	if _, c := ready().Update(tui.Key{Type: tui.KeyRunes, Text: "q"}); c == nil {
		t.Fatal("q did not quit")
	}
}

func TestHoverShowsThePosition(t *testing.T) {
	m := ready()
	x, y := px(t, m, 7, 3)
	m = mouse(m, x, y, tui.MouseButtonNone, tui.MouseActionMotion)
	if !strings.Contains(text(m), "7,3") {
		t.Fatalf("no position:\n%s", text(m))
	}
}
