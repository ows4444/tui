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

func ready100() model { return send(newModel(), tui.ResizeMsg{Width: 100, Height: 30}) }

func text(m model) string { return ansi.StripANSI(m.View()) }

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

func click(m model, x, y int) (model, tui.Cmd) {
	return update(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

func tickOf(m model) tui.Msg { return tickMsg{m.seq} }

func TestFirstScreenShowsTheArenaScoreAndAPrompt(t *testing.T) {
	out := text(ready100())
	for _, want := range []string{"snake", "score 0", "best 0", "speed 2/5", "[pause]", "[restart]", "Press an arrow key", "┌", "┘", "██", "▓▓", "◆"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestArenaFitsTheTerminal(t *testing.T) {
	m := send(newModel(), tui.ResizeMsg{Width: 60, Height: 14})
	if m.g.w != 29 || m.g.h != 9 {
		t.Fatalf("arena %dx%d", m.g.w, m.g.h)
	}
	if !m.fits() {
		t.Fatal("does not fit")
	}
	m = send(newModel(), tui.ResizeMsg{Width: 300, Height: 80})
	if m.g.w != maxW || m.g.h != maxH {
		t.Fatalf("arena %dx%d", m.g.w, m.g.h)
	}
}

func TestAnArrowStartsTheGameAndATickMovesTheSnake(t *testing.T) {
	m, cmd := key(ready100(), tui.KeyUp)
	if m.g.state != running || cmd == nil {
		t.Fatalf("state %v cmd %v", m.g.state, cmd)
	}
	head := m.g.body[0]
	m, cmd = update(m, tickOf(m))
	if m.g.body[0] != (pt{head.x, head.y - 1}) || cmd == nil {
		t.Fatalf("head %v after a tick, cmd %v", m.g.body[0], cmd)
	}
}

func TestStaleTicksAreIgnored(t *testing.T) {
	m, _ := key(ready100(), tui.KeyUp)
	stale := tickMsg{m.seq - 1}
	head := m.g.body[0]
	m, cmd := update(m, stale)
	if m.g.body[0] != head || cmd != nil {
		t.Fatal("a stale tick moved the snake or restarted the timer")
	}
}

func TestPauseStopsTheClockAndResumeStartsANewOne(t *testing.T) {
	m, _ := key(ready100(), tui.KeyRight)
	old := tickOf(m)
	m, _ = runes(m, "p")
	if m.g.state != paused || !strings.Contains(text(m), "Paused") || !strings.Contains(text(m), "[resume]") {
		t.Fatalf("not paused:\n%s", text(m))
	}
	head := m.g.body[0]
	m = send(m, old)
	if m.g.body[0] != head {
		t.Fatal("the paused snake moved")
	}
	m, cmd := runes(m, " ")
	if m.g.state != running || cmd == nil {
		t.Fatal("space did not resume")
	}
}

func TestTheSnakeDiesOnTheWallAndRestartPlaysAgain(t *testing.T) {
	m, _ := key(ready100(), tui.KeyRight)
	for i := 0; i < 60 && m.g.state == running; i++ {
		m = send(m, tickOf(m))
	}
	if m.g.state != over || !strings.Contains(text(m), "Game over") {
		t.Fatalf("state %v:\n%s", m.g.state, text(m))
	}
	m, _ = runes(m, "r")
	if m.g.state != ready || m.g.score != 0 || len(m.g.body) != 3 {
		t.Fatalf("restart: state %v", m.g.state)
	}
}

func TestTheBestScoreSurvivesARestart(t *testing.T) {
	m, _ := key(ready100(), tui.KeyRight)
	m.g.food = pt{m.g.body[0].x + 1, m.g.body[0].y}
	m = send(m, tickOf(m))
	if m.g.score != 2 || m.best != 2 {
		t.Fatalf("score %d best %d", m.g.score, m.best)
	}
	m, _ = runes(m, "r")
	if m.g.score != 0 || m.best != 2 || !strings.Contains(text(m), "best 2") {
		t.Fatalf("score %d best %d", m.g.score, m.best)
	}
}

func TestSpeedKeysAndButtons(t *testing.T) {
	m, _ := runes(ready100(), "+")
	if m.level != 2 || !strings.Contains(text(m), "speed 3/5") {
		t.Fatalf("level %d", m.level)
	}
	for i := 0; i < 6; i++ {
		m, _ = runes(m, "+")
	}
	if m.level != 4 {
		t.Fatalf("level %d past the top", m.level)
	}
	r := rect(t, m, "slower")
	m, _ = click(m, r.X, r.Y)
	if m.level != 3 {
		t.Fatalf("slower: level %d", m.level)
	}
	r = rect(t, m, "faster")
	if m, _ = click(m, r.X, r.Y); m.level != 4 {
		t.Fatalf("faster: level %d", m.level)
	}
}

func TestClickingTheArenaSteersTowardThePointer(t *testing.T) {
	m := ready100()
	a := rect(t, m, "arena")
	head := m.g.body[0]
	m, cmd := click(m, a.X+2*head.x+1, a.Y) // straight above the head
	if m.g.state != running || m.g.next != up || cmd == nil {
		t.Fatalf("state %v next %v", m.g.state, m.g.next)
	}
	m, _ = click(m, a.X+2*head.x+1, a.Y+a.H-1) // below: allowed, the snake still heads right
	if m.g.next != down {
		t.Fatalf("next %v", m.g.next)
	}
}

func TestPauseAndRestartButtons(t *testing.T) {
	m, _ := key(ready100(), tui.KeyRight)
	r := rect(t, m, "pause")
	m, _ = click(m, r.X+1, r.Y)
	if m.g.state != paused {
		t.Fatalf("state %v", m.g.state)
	}
	r = rect(t, m, "restart")
	if m, _ = click(m, r.X+1, r.Y); m.g.state != ready {
		t.Fatalf("state %v", m.g.state)
	}
}

func TestTooSmallWindowPausesAndSaysSo(t *testing.T) {
	m, _ := key(ready100(), tui.KeyRight)
	m = send(m, tui.ResizeMsg{Width: 30, Height: 8})
	if m.g.state != paused || !strings.Contains(text(m), "too small") {
		t.Fatalf("state %v:\n%s", m.g.state, text(m))
	}
	if m, _ = runes(m, "p"); m.g.state != paused || !strings.Contains(m.msg, "bigger") {
		t.Fatalf("resumed in a window that is too small: %q", m.msg)
	}
}

func TestHelpPausesAndCloses(t *testing.T) {
	m, _ := key(ready100(), tui.KeyRight)
	m, _ = runes(m, "?")
	if !m.help || m.g.state != paused || !strings.Contains(text(m), "turn toward the pointer") {
		t.Fatalf("no help:\n%s", text(m))
	}
	if m, _ = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
}

func TestQuit(t *testing.T) {
	for _, quit := range []func() tui.Cmd{
		func() tui.Cmd { _, c := key(ready100(), tui.KeyCtrlC); return c },
		func() tui.Cmd { _, c := runes(ready100(), "q"); return c },
	} {
		cmd := quit()
		if cmd == nil {
			t.Fatal("no quit command")
		}
		if _, ok := cmd().(tui.QuitMsg); !ok {
			t.Fatal("not a quit")
		}
	}
}
