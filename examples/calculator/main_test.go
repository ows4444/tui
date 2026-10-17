package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func send(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func typeKey(m model, s string) model {
	m, _ = send(m, tui.Key{Type: tui.KeyRunes, Text: s})
	return m
}

func typeAll(m model, s string) model {
	for _, r := range s {
		m = typeKey(m, string(r))
	}
	return m
}

func centre(t testing.TB, m model, id string) (x, y int) {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no target %q on screen", id)
	}
	return r.X + r.W/2, r.Y + r.H/2
}

func click(t testing.TB, m model, id string) (model, tui.Cmd) {
	t.Helper()
	x, y := centre(t, m, id)
	m, _ = send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	return send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease})
}

func clickKeys(t testing.TB, m model, labels ...string) model {
	t.Helper()
	for _, l := range labels {
		m, _ = click(t, m, "k-"+l)
	}
	return m
}

func TestClickingButtonsCalculates(t *testing.T) {
	m := clickKeys(t, initialModel(), "1", "2", "+", "8", "×", "2", "=")
	if m.expr != "28" || !m.done {
		t.Fatalf("expr = %q done=%v, want 28", m.expr, m.done)
	}
	if len(m.history) != 1 || m.history[0] != (entry{"12+8×2", "28"}) {
		t.Fatalf("history = %v", m.history)
	}
}

func TestKeyboardCalculates(t *testing.T) {
	m := typeAll(initialModel(), "(1+2)*3-4/2")
	m, _ = send(m, tui.Key{Type: tui.KeyEnter})
	if m.expr != "7" {
		t.Fatalf("expr = %q, want 7", m.expr)
	}
}

func TestPreviewShowsBeforeEquals(t *testing.T) {
	m := typeAll(initialModel(), "6*7")
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "= 42") {
		t.Fatalf("no live preview:\n%s", out)
	}
}

func TestErrorsAreShownAndClearedByTheNextKey(t *testing.T) {
	m := typeAll(initialModel(), "1/0")
	m, _ = send(m, tui.Key{Type: tui.KeyEnter})
	if m.err == "" || !strings.Contains(ansi.StripANSI(m.View()), "cannot divide by zero") {
		t.Fatalf("no error shown: %q", m.err)
	}
	m = typeKey(m, "5")
	if m.err != "" {
		t.Fatal("the error stayed after the next key")
	}
}

func TestBackspaceAndClear(t *testing.T) {
	m := typeAll(initialModel(), "123")
	m, _ = send(m, tui.Key{Type: tui.KeyBackspace})
	if m.expr != "12" {
		t.Fatalf("expr = %q after Backspace", m.expr)
	}
	m = clickKeys(t, m, "C")
	if m.expr != "" {
		t.Fatalf("expr = %q after C", m.expr)
	}
}

func TestOperatorNeedsSomethingBeforeIt(t *testing.T) {
	m := typeKey(initialModel(), "+")
	if m.expr != "" {
		t.Fatalf("expr = %q: a leading + must be ignored", m.expr)
	}
	m = typeKey(m, "-")
	if m.expr != "−" {
		t.Fatalf("expr = %q: a leading minus must be allowed", m.expr)
	}
}

func TestDigitAfterResultStartsAfresh_OperatorContinues(t *testing.T) {
	m := typeAll(initialModel(), "2+2")
	m, _ = send(m, tui.Key{Type: tui.KeyEnter})
	cont := typeAll(m, "*3")
	cont, _ = send(cont, tui.Key{Type: tui.KeyEnter})
	if cont.expr != "12" {
		t.Fatalf("4*3 gave %q", cont.expr)
	}
	fresh := typeKey(m, "9")
	if fresh.expr != "9" {
		t.Fatalf("a digit after a result gave %q, want 9", fresh.expr)
	}
}

func TestHistoryClickInsertsTheResult(t *testing.T) {
	m := typeAll(initialModel(), "5*5")
	m, _ = send(m, tui.Key{Type: tui.KeyEnter})
	m = typeKey(m, "1") // a fresh expression
	m, _ = click(t, m, "hist-0")
	if m.expr != "125" {
		t.Fatalf("expr = %q, want 1 then the history result 25", m.expr)
	}
}

func TestHistoryIsBounded(t *testing.T) {
	m := initialModel()
	for i := 0; i < historyMax+4; i++ {
		m = typeAll(m, "1+1")
		m, _ = send(m, tui.Key{Type: tui.KeyEnter})
		m = typeKey(m, "C")
	}
	if len(m.history) != historyMax {
		t.Fatalf("history holds %d, want %d", len(m.history), historyMax)
	}
}

func TestCopyWritesTheResult(t *testing.T) {
	m := typeAll(initialModel(), "2+2")
	m, cmd := click(t, m, "copy")
	if cmd == nil || !m.copied {
		t.Fatal("clicking the display did not copy")
	}
	if _, cmd := send(initialModel(), tui.Key{Type: tui.KeyRunes, Text: "y"}); cmd != nil {
		t.Fatal("y with nothing to copy returned a Cmd")
	}
}

func TestReleaseOffTheButtonCancels(t *testing.T) {
	m := initialModel()
	x, y := centre(t, m, "k-7")
	m, _ = send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	m, _ = send(m, tui.MouseEvent{X: 0, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease})
	if m.expr != "" {
		t.Fatalf("expr = %q: dragging off must cancel the click", m.expr)
	}
}

func TestHover(t *testing.T) {
	m := initialModel()
	x, y := centre(t, m, "k-5")
	m, _ = send(m, tui.MouseEvent{X: x, Y: y, Action: tui.MouseActionMotion})
	if m.hover != "k-5" {
		t.Fatalf("hover = %q", m.hover)
	}
}

func TestNarrowTerminalDropsTheHistory(t *testing.T) {
	m, _ := send(initialModel(), tui.ResizeMsg{Width: 40, Height: 24})
	if _, ok := layout.RectOf(m.screen(), m.size(), "hist-0"); ok || m.wide() {
		t.Fatal("history shown at 40 columns")
	}
	if !strings.Contains(ansi.StripANSI(initialModel().View()), "History") {
		t.Fatal("history missing at 80 columns")
	}
}

func TestQuit(t *testing.T) {
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyRunes, Text: "q"}, {Type: tui.KeyEsc}} {
		if _, cmd := send(initialModel(), k); !isQuit(cmd) {
			t.Errorf("%v did not quit", k)
		}
	}
	m := typeKey(initialModel(), "4")
	m, cmd := send(m, tui.Key{Type: tui.KeyEsc})
	if isQuit(cmd) || m.expr != "" {
		t.Fatal("Esc with an expression must clear it, not quit")
	}
}
