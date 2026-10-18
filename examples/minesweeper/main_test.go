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

func key(m model, t tui.KeyType) (model, tui.Cmd) { return update(m, tui.Key{Type: t}) }

func runes(m model, s string) (model, tui.Cmd) {
	return update(m, tui.Key{Type: tui.KeyRunes, Text: s})
}

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

func at(t testing.TB, m model, x, y int) (int, int) {
	r := rect(t, m, "board")
	return r.X + 2*x, r.Y + y
}

func press(m model, x, y int, b tui.MouseButton, clicks int) (model, tui.Cmd) {
	return update(m, tui.MouseEvent{X: x, Y: y, Button: b, Action: tui.MouseActionPress, Clicks: clicks})
}

// withMines swaps in a known board, started, with the clock not running.
func withMines(m model, rows ...string) model {
	m.g = board(rows...)
	return m
}

func TestFirstScreen(t *testing.T) {
	out := text(ready())
	for _, want := range []string{"minesweeper", "mines 10", "time 0", ":) new", "1 9x9", "2 16x16", "3 30x16", "Beginner: 10 mines. The first click is safe", "■", "┌", "┘"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestSpaceOpensAndStartsTheClock(t *testing.T) {
	m, cmd := runes(ready(), " ")
	if m.g.state != playing || cmd == nil {
		t.Fatalf("state %v cmd %v", m.g.state, cmd)
	}
	if m.g.at(m.cx, m.cy).open != true {
		t.Fatal("the cursor cell is closed")
	}
	m, cmd = update(m, tickMsg{m.seq})
	if m.secs != 1 || cmd == nil || !strings.Contains(text(m), "time 1") {
		t.Fatalf("secs %d", m.secs)
	}
	m, cmd = update(m, tickMsg{m.seq - 1})
	if m.secs != 1 || cmd != nil {
		t.Fatal("a stale tick counted")
	}
}

func TestCursorKeysAndFlagKey(t *testing.T) {
	m, _ := key(ready(), tui.KeyRight)
	m, _ = key(m, tui.KeyDown)
	m, _ = runes(m, "l")
	if m.cx != 2 || m.cy != 1 {
		t.Fatalf("cursor %d,%d", m.cx, m.cy)
	}
	for i := 0; i < 20; i++ {
		m, _ = runes(m, "h")
		m, _ = runes(m, "k")
	}
	if m.cx != 0 || m.cy != 0 {
		t.Fatalf("cursor %d,%d past the edge", m.cx, m.cy)
	}
	m, _ = runes(m, "f")
	if !m.g.at(0, 0).flag || !strings.Contains(text(m), "mines 9") || !strings.Contains(text(m), "⚑") {
		t.Fatalf("no flag:\n%s", text(m))
	}
}

func TestLeftRightAndMiddleClicks(t *testing.T) {
	m := withMines(ready(), "*..", "...", "...")
	x, y := at(t, m, 2, 2)
	m, _ = press(m, x, y, tui.MouseButtonRight, 1)
	if !m.g.at(2, 2).flag {
		t.Fatal("right click did not flag")
	}
	m, _ = press(m, x, y, tui.MouseButtonLeft, 1)
	if m.g.at(2, 2).open {
		t.Fatal("left click opened a flagged cell")
	}
	m, _ = press(m, x, y, tui.MouseButtonRight, 1)
	x1, y1 := at(t, m, 1, 1)
	m, _ = press(m, x1, y1, tui.MouseButtonLeft, 1)
	if !m.g.at(1, 1).open {
		t.Fatal("left click did not open")
	}
	fx, fy := at(t, m, 0, 0)
	m, _ = press(m, fx, fy, tui.MouseButtonRight, 1)
	m, _ = press(m, x1, y1, tui.MouseButtonMiddle, 1)
	if m.g.state != won {
		t.Fatalf("middle click did not chord: %v\n%s", m.g.state, text(m))
	}
	if !strings.Contains(m.msg, "Cleared in") {
		t.Fatalf("msg %q", m.msg)
	}
}

func TestDoubleClickChordsAndBoomEndsTheGame(t *testing.T) {
	m := withMines(ready(), "*..", "...", "...")
	x, y := at(t, m, 1, 1)
	m, _ = press(m, x, y, tui.MouseButtonLeft, 1)
	fx, fy := at(t, m, 1, 0)
	m, _ = press(m, fx, fy, tui.MouseButtonRight, 1) // a wrong flag
	m, _ = press(m, x, y, tui.MouseButtonLeft, 2)
	if m.g.state != lost || !strings.Contains(text(m), "Boom") || !strings.Contains(text(m), ":(") {
		t.Fatalf("state %v:\n%s", m.g.state, text(m))
	}
	if !strings.Contains(text(m), "x ") || !strings.Contains(text(m), "* ") {
		t.Fatalf("a lost board shows the mines and the wrong flag:\n%s", text(m))
	}
	m, cmd := press(m, x, y, tui.MouseButtonLeft, 1)
	if cmd != nil || m.g.state != lost {
		t.Fatal("clicks after the end changed the game")
	}
}

func TestSizeButtonsAndKeysStartAFreshGame(t *testing.T) {
	m, _ := runes(ready(), "3")
	if m.lv != 2 || m.g.lv.w != 30 || !strings.Contains(text(m), "mines 99") {
		t.Fatalf("level %d", m.lv)
	}
	r := rect(t, m, "size-1")
	m, _ = press(m, r.X+1, r.Y, tui.MouseButtonLeft, 1)
	if m.lv != 1 || m.g.lv.w != 16 {
		t.Fatalf("level %d", m.lv)
	}
	m, _ = runes(m, " ")
	r = rect(t, m, "new")
	if m, _ = press(m, r.X+1, r.Y, tui.MouseButtonLeft, 1); m.g.state != fresh || m.secs != 0 {
		t.Fatalf("state %v secs %d", m.g.state, m.secs)
	}
}

func TestNewGameStopsTheOldClock(t *testing.T) {
	m, _ := runes(ready(), " ")
	old := tickMsg{m.seq}
	m, _ = runes(m, "n")
	if m, cmd := update(m, old); m.secs != 0 || cmd != nil {
		t.Fatal("the old timer kept counting")
	}
}

func TestTooSmallWindowSaysWhatItNeeds(t *testing.T) {
	m := send(ready(), tui.ResizeMsg{Width: 40, Height: 10})
	if !strings.Contains(text(m), "needs 20x14") {
		t.Fatalf("\n%s", text(m))
	}
	m, _ = runes(m, "3")
	if !strings.Contains(text(m), "needs 62x21") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestHelpAndQuit(t *testing.T) {
	m, _ := runes(ready(), "?")
	if !strings.Contains(text(m), "open around a number") {
		t.Fatalf("no help:\n%s", text(m))
	}
	if m, _ = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	if _, c := key(ready(), tui.KeyCtrlC); c == nil {
		t.Fatal("Ctrl+C did not quit")
	}
	if _, c := runes(ready(), "q"); c == nil {
		t.Fatal("q did not quit")
	}
}

func TestHoverMarksTheCellUnderThePointer(t *testing.T) {
	m := ready()
	x, y := at(t, m, 3, 3)
	m = send(m, tui.MouseEvent{X: x, Y: y, Action: tui.MouseActionMotion})
	if m.hx != 3 || m.hy != 3 {
		t.Fatalf("hover %d,%d", m.hx, m.hy)
	}
	m = send(m, tui.MouseEvent{X: 0, Y: 29, Action: tui.MouseActionMotion})
	if m.hx != -1 {
		t.Fatalf("hover %d,%d off the board", m.hx, m.hy)
	}
}
