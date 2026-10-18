package main

import (
	"errors"
	"strings"
	"testing"
	"time"

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

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func ready() model {
	m := newModel()
	t := time.Unix(100, 0)
	m.now = func() time.Time { t = t.Add(10 * time.Millisecond); return t }
	return send(m, tui.ResizeMsg{Width: 120, Height: 30})
}

func text(m model) string { return ansi.StripANSI(m.View()) }

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

func motion(m model, x, y int) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Action: tui.MouseActionMotion})
}

func TestFirstScreenWaitsForInput(t *testing.T) {
	m := send(newModel(), tui.ResizeMsg{Width: 120, Height: 30})
	out := text(m)
	for _, want := range []string{"inputlab", "● keys", "● mouse", "● paste", "● other", "join motion", "resize  120x30", "Last key", "probing", "[ctrl+c] quit"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestKeysAreDecodedWithTheirFields(t *testing.T) {
	m := send(ready(), tui.Key{Type: tui.KeyRunes, Text: "é", Code: 'é'})
	m = send(m, tui.Key{Type: tui.KeyUp, Mod: input.ModCtrl | input.ModAlt})
	m = send(m, tui.Key{Type: tui.KeyCtrl, Code: 'a'})
	out := text(m)
	for _, want := range []string{`text="é"`, "ctrl+alt+up", "mod=ctrl+alt", "ctrl+a", "mod=-"} {
		if !strings.Contains(out, want) {
			t.Errorf("log lacks %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "press") {
		t.Fatal("no action")
	}
}

func TestReleaseAndRepeatAreLogged(t *testing.T) {
	m := send(ready(), tui.KeyRepeatMsg{Key: tui.Key{Type: tui.KeyRunes, Text: "a", Code: 'a'}})
	m = send(m, tui.KeyReleaseMsg{Key: tui.Key{Type: tui.KeyRunes, Text: "a", Code: 'a'}})
	out := text(m)
	if !strings.Contains(out, "repeat") || !strings.Contains(out, "release") {
		t.Fatalf("\n%s", out)
	}
	if !strings.Contains(m.lastKey, "release") {
		t.Fatalf("last key %q", m.lastKey)
	}
}

func TestMouseEventsShowButtonActionAndPlace(t *testing.T) {
	m := send(ready(), tui.MouseEvent{X: 7, Y: 3, Button: tui.MouseButtonRight, Action: tui.MouseActionPress, Clicks: 1, Mod: input.ModShift})
	m = send(m, tui.MouseEvent{X: 7, Y: 3, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	out := text(m)
	for _, want := range []string{"press", "right", "at 7,3", "mod=shift", "clicks=1", "wheel-down"} {
		if !strings.Contains(out, want) {
			t.Errorf("log lacks %q:\n%s", want, out)
		}
	}
}

func TestMotionIsJoinedUnlessTurnedOff(t *testing.T) {
	m := ready()
	for x := 0; x < 20; x++ {
		m = motion(m, x, 4)
	}
	n := len(m.log)
	last := m.log[n-1]
	if last.count != 20 || !strings.Contains(last.text, "at 19,4") || m.counts[kMouse] != 20 {
		t.Fatalf("count %d text %q mouse %d", last.count, last.text, m.counts[kMouse])
	}
	if !strings.Contains(text(m), "×20") {
		t.Fatalf("no repeat count:\n%s", text(m))
	}
	m = key(m, tui.KeyF6)
	before := len(m.log)
	m = motion(motion(m, 1, 1), 2, 1)
	if len(m.log) != before+2 {
		t.Fatalf("motion should log one line each, got %d", len(m.log)-before)
	}
	m = send(m, tui.MouseEvent{X: 2, Y: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: 1})
	m = motion(m, 3, 1)
	if m.log[len(m.log)-1].count != 1 {
		t.Fatal("motion joined across a click")
	}
}

func TestPasteFocusSchemeAndErrorLines(t *testing.T) {
	m := send(ready(), tui.PasteEvent{Text: "line one\nline two\tend"})
	m = send(m, tui.PasteEvent{Text: "x", Incomplete: true, Truncated: true})
	m = send(m, tui.FocusEvent{Focused: false})
	m = send(m, tui.ColorSchemeEvent{Dark: false})
	m = send(m, tui.InputErrorMsg{Err: errors.New("read: closed")})
	out := text(m)
	for _, want := range []string{"paste   21 bytes, 2 lines", "line␠one⏎", "⇥", "incomplete truncated", "focus   out", "scheme  light", "read: closed", "focus   out"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if m.focus != "out" || m.scheme != "light" {
		t.Fatalf("focus %q scheme %q", m.focus, m.scheme)
	}
}

func TestCapabilitiesReplaceTheProbingLine(t *testing.T) {
	m := send(ready(), tui.CapabilitiesMsg{Capabilities: tui.Capabilities{KittyKeyboard: true, Sixel: true, XTVersion: "kitty(0.35.2)"}})
	out := text(m)
	for _, want := range []string{"kitty keys      yes", "sixel           yes", "kitty graphics  no", "kitty(0.35.2)", "probe finished"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "probing") {
		t.Fatal("still probing")
	}
}

func TestFiltersByKeyAndClick(t *testing.T) {
	m := send(ready(), tui.PasteEvent{Text: "hello"})
	m = key(m, tui.KeyF4)
	if m.shown[kPaste] || strings.Contains(text(m), "   2 paste") || !strings.Contains(text(m), "○ paste") {
		t.Fatalf("paste still shown:\n%s", text(m))
	}
	r := rect(t, m, "kind-2")
	m = send(m, tui.MouseEvent{X: r.X + 1, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: 1})
	if !m.shown[kPaste] || !strings.Contains(text(m), "   2 paste") {
		t.Fatalf("click did not show paste:\n%s", text(m))
	}
	m = key(key(m, tui.KeyF2), tui.KeyF3)
	if m.shown[kKey] || m.shown[kMouse] {
		t.Fatal("F2/F3 did not hide")
	}
	if m = key(m, tui.KeyF5); m.shown[kOther] {
		t.Fatal("F5 did not hide other")
	}
}

func TestPauseFreezesTheViewButKeepsCounting(t *testing.T) {
	m := send(ready(), tui.PasteEvent{Text: "before"})
	m = send(m, tui.Key{Type: tui.KeyCtrl, Code: 'p'})
	if !m.paused {
		t.Fatal("not paused")
	}
	m = send(m, tui.PasteEvent{Text: "after-the-pause"})
	out := text(m)
	if strings.Contains(out, "after-the-pause") || !strings.Contains(out, "1 new events since the pause") && !strings.Contains(out, "new events since") {
		t.Fatalf("\n%s", out)
	}
	if m.counts[kPaste] != 2 {
		t.Fatalf("paste count %d", m.counts[kPaste])
	}
	r := rect(t, m, "pause")
	m = send(m, tui.MouseEvent{X: r.X + 1, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: 1})
	if m.paused || !strings.Contains(text(m), "after-the-pause") {
		t.Fatalf("resume failed:\n%s", text(m))
	}
}

func TestPageUpScrollsBackAndPauses(t *testing.T) {
	m := ready()
	for i := 0; i < 80; i++ {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(rune('a' + i%26)), Code: rune('a' + i%26)})
	}
	if !strings.Contains(text(m), "  82 ") && !strings.Contains(text(m), "  81 ") {
		t.Fatalf("newest lines not shown:\n%s", text(m))
	}
	m = key(m, tui.KeyPgUp)
	if !m.paused || m.back == 0 || !strings.Contains(text(m), "lines back") {
		t.Fatalf("paused %v back %d", m.paused, m.back)
	}
	back := m.back
	m = key(m, tui.KeyPgDown)
	if m.back >= back {
		t.Fatalf("PgDn did not come forward: %d", m.back)
	}
	for i := 0; i < 20; i++ {
		m = key(m, tui.KeyPgUp)
	}
	if m.back > len(m.visible(true))-m.logRows() {
		t.Fatalf("scrolled past the first line: %d", m.back)
	}
}

func TestClearEmptiesLogAndCounters(t *testing.T) {
	m := send(ready(), tui.PasteEvent{Text: "x"})
	m = send(m, tui.Key{Type: tui.KeyCtrl, Code: 'l'})
	if len(m.log) != 0 || m.total != 0 || m.counts[kPaste] != 0 {
		t.Fatalf("log %d total %d", len(m.log), m.total)
	}
	m = send(m, tui.PasteEvent{Text: "y"})
	r := rect(t, m, "clear")
	m = send(m, tui.MouseEvent{X: r.X, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: 1})
	if len(m.log) != 0 {
		t.Fatalf("click on clear: %+v", m.log)
	}
}

func TestLogIsCappedAndGapsAreMeasured(t *testing.T) {
	m := ready()
	for range maxLog + 50 {
		m = send(m, tui.PasteEvent{Text: "x"})
	}
	if len(m.log) != maxLog || m.total != maxLog+51 {
		t.Fatalf("log %d total %d", len(m.log), m.total)
	}
	if g := m.log[len(m.log)-1].gap; g != 10*time.Millisecond || !strings.Contains(text(m), "last gap 10ms") {
		t.Fatalf("gap %v", g)
	}
}

func TestHelpShowsReservedKeysAndStillLogs(t *testing.T) {
	m := key(ready(), tui.KeyF1)
	out := text(m)
	if !m.help || !strings.Contains(out, "pause or resume the log") || !strings.Contains(out, "kitty keyboard protocol") {
		t.Fatalf("no help:\n%s", out)
	}
	m = send(m, tui.Key{Type: tui.KeyCtrl, Code: 'l'})
	if len(m.log) == 0 {
		t.Fatal("ctrl+l cleared the log while help was open")
	}
	if m = key(m, tui.KeyF1); m.help {
		t.Fatal("F1 did not close help")
	}
}

func TestQuitIsCtrlCAndQIsJustAKey(t *testing.T) {
	m, cmd := update(ready(), tui.Key{Type: tui.KeyCtrlC})
	if cmd == nil || !strings.Contains(text(m), "ctrl+c") {
		t.Fatal("Ctrl+C should quit and be logged")
	}
	m, cmd = update(ready(), tui.Key{Type: tui.KeyRunes, Text: "q", Code: 'q'})
	if cmd != nil || !strings.Contains(text(m), `text="q"`) {
		t.Fatal("q should only be logged")
	}
}

func TestNarrowWindowsDropTheSidePanel(t *testing.T) {
	m := send(ready(), tui.ResizeMsg{Width: 80, Height: 24})
	if strings.Contains(text(m), "Last key") {
		t.Fatal("side panel at 80 columns")
	}
	if !strings.Contains(text(send(m, tui.ResizeMsg{Width: 90, Height: 24})), "Last key") {
		t.Fatal("no side panel at 90 columns")
	}
}
