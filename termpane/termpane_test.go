//go:build darwin || linux

package termpane

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
)

// pane starts a pane running sh -c script at cols x rows and returns it with
// the Cmd that listens for its next message. The pane is closed when the test
// ends.
func pane(t *testing.T, cols, rows int, script string) (Model, tui.Cmd) {
	t.Helper()
	m := New("sh", "-c", script)
	m.ID = "p"
	m.Cols, m.Rows = cols, rows
	cmd := m.Init()
	t.Cleanup(func() { _ = m.Close() })
	return m, cmd
}

// next runs cmd and gives its message, failing the test if none comes soon.
func next(t *testing.T, cmd tui.Cmd) tui.Msg {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	msg := tui.RunCmd(ctx, cmd)
	if msg == nil {
		t.Fatal("no message from the pane")
	}
	return msg
}

// until feeds the pane's messages to Update until ok(m) holds, and returns the
// pane, the Cmd that listens next and the exit message if one came.
func until(t *testing.T, m Model, cmd tui.Cmd, ok func(Model) bool) (Model, tui.Cmd, *ExitedMsg) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if ok(m) {
			return m, cmd, nil
		}
		if cmd == nil {
			t.Fatal("the pane stopped listening before the condition held")
		}
		msg := next(t, cmd)
		if ex, isExit := msg.(ExitedMsg); isExit {
			if ok(m) {
				return m, nil, &ex
			}
			return m, nil, &ex
		}
		m, cmd = m.Update(msg)
	}
	t.Fatalf("condition not met; screen:\n%s", text(m))
	return m, nil, nil
}

func text(m Model) string { return ansi.StripANSI(m.View()) }

func has(s string) func(Model) bool {
	return func(m Model) bool { return strings.Contains(text(m), s) }
}

func runeKey(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r} }

func TestShowsTheOutputAndTheExitStatus(t *testing.T) {
	m, cmd := pane(t, 20, 4, `printf 'hello\nworld\n'; exit 3`)
	m, _, ex := until(t, m, cmd, func(Model) bool { return false }) // run to the end
	if ex == nil {
		t.Fatal("no ExitedMsg")
	}
	if ex.Code != 3 || ex.Err != nil || ex.ID != "p" {
		t.Fatalf("ExitedMsg = %+v", *ex)
	}
	rows := strings.Split(text(m), "\n")
	if len(rows) != 4 || strings.TrimSpace(rows[0]) != "hello" || strings.TrimSpace(rows[1]) != "world" {
		t.Fatalf("screen:\n%s", text(m))
	}
	if m.Running() {
		t.Error("Running after the exit")
	}
}

func TestKeysReachTheProgram(t *testing.T) {
	m, cmd := pane(t, 30, 5, `cat`)
	m.Focus()
	for _, k := range []tui.Key{runeKey('h'), runeKey('i'), {Type: tui.KeyEnter}} {
		m, _ = m.Update(k)
	}
	// The terminal echoes the typed line and cat prints it back: twice on screen.
	m, cmd, _ = until(t, m, cmd, func(m Model) bool { return strings.Count(text(m), "hi") >= 2 })
	m, _ = m.Update(tui.Key{Type: tui.KeyCtrl, Code: 'd', Mod: input.ModCtrl})
	_, _, ex := until(t, m, cmd, func(Model) bool { return false })
	if ex == nil || ex.Code != 0 {
		t.Fatalf("cat did not end on Ctrl+D: %+v", ex)
	}
}

func TestUnfocusedPaneSendsNothing(t *testing.T) {
	m, cmd := pane(t, 30, 3, `cat`)
	m, _ = m.Update(runeKey('x'))
	m, _ = m.Update(tui.PasteEvent{Text: "pasted"})
	m.Input("typed\n") // Input is the application's own, and always goes through
	m, _, _ = until(t, m, cmd, has("typed"))
	if out := text(m); strings.Contains(out, "x") || strings.Contains(out, "pasted") {
		t.Fatalf("an unfocused pane sent input:\n%s", out)
	}
}

func TestSetSizeResizesTheProgramsTerminal(t *testing.T) {
	m, cmd := pane(t, 40, 6, `read x; stty size`)
	m.SetSize(30, 10)
	if c, r := m.Size(); c != 30 || r != 10 {
		t.Fatalf("Size = %d x %d", c, r)
	}
	m.Focus()
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	until(t, m, cmd, has("10 30"))
	if rows := strings.Split(text(m), "\n"); len(rows) != 10 {
		t.Fatalf("%d rows", len(rows))
	}
}

func TestEnvironmentAndColours(t *testing.T) {
	m, cmd := pane(t, 40, 3, `echo "$TERM $COLORTERM"; printf '\033[1;31mred\033[0m plain'`)
	m, _, _ = until(t, m, cmd, has("red plain"))
	if !strings.Contains(text(m), "xterm-256color truecolor") {
		t.Fatalf("screen:\n%s", text(m))
	}
	if v := m.View(); !strings.Contains(v, "\x1b[1;31mred") && !strings.Contains(v, "\x1b[31;1mred") && !strings.Contains(v, "31m") {
		t.Fatalf("the colour was lost: %q", v)
	}
	// An Env of the caller's own is not overridden.
	e := New("sh", "-c", `echo $TERM`)
	e.Env = []string{"TERM=vt100", "PATH=" + "/usr/bin:/bin"}
	got := strings.Join(e.env(), " ")
	if !strings.Contains(got, "TERM=vt100") || strings.Contains(got, "xterm-256color") {
		t.Fatalf("env = %q", got)
	}
}

func TestScrollbackAndTheBadge(t *testing.T) {
	m, cmd := pane(t, 20, 4, `i=1; while [ $i -le 30 ]; do echo line$i; i=$((i+1)); done`)
	m, _, _ = until(t, m, cmd, func(Model) bool { return false })
	m.Focus()
	if !strings.Contains(text(m), "line30") || strings.Contains(text(m), "line1\n") {
		t.Fatalf("live screen:\n%s", text(m))
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyPgUp, Mod: input.ModShift})
	if m.Scrolled() == 0 {
		t.Fatal("Shift+PgUp did not scroll back")
	}
	if !strings.Contains(text(m), "scrollback") {
		t.Fatalf("no badge:\n%s", text(m))
	}
	if strings.Contains(text(m), "line30") {
		t.Fatalf("still showing the end:\n%s", text(m))
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyPgDown, Mod: input.ModShift})
	m, _ = m.Update(tui.Key{Type: tui.KeyPgUp, Mod: input.ModShift})
	m, _ = m.Update(tui.Key{Type: tui.KeyEnd, Mod: input.ModShift})
	if m.Scrolled() != 0 {
		t.Fatalf("Shift+End left the view at %d", m.Scrolled())
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyPgUp, Mod: input.ModShift})
	m, _ = m.Update(runeKey('q')) // typing returns to the live screen
	if m.Scrolled() != 0 {
		t.Fatalf("a key left the view at %d", m.Scrolled())
	}
}

func TestWheelScrollsBackWhenTheProgramDoesNotWantTheMouse(t *testing.T) {
	m, cmd := pane(t, 20, 3, `i=1; while [ $i -le 20 ]; do echo line$i; i=$((i+1)); done`)
	m, _, _ = until(t, m, cmd, func(Model) bool { return false })
	m.Mouse, m.Bounds = true, hittest.Rect{X: 5, Y: 2, W: 20, H: 3}
	wheel := func(b tui.MouseButton, x, y int) tui.MouseEvent {
		return tui.MouseEvent{X: x, Y: y, Button: b, Action: tui.MouseActionPress}
	}
	m, _ = m.Update(wheel(tui.MouseButtonWheelUp, 6, 3))
	if m.Scrolled() != 3 {
		t.Fatalf("scrolled %d", m.Scrolled())
	}
	m, _ = m.Update(wheel(tui.MouseButtonWheelUp, 0, 0)) // outside the pane
	if m.Scrolled() != 3 {
		t.Fatalf("an event outside the pane scrolled it to %d", m.Scrolled())
	}
	m, _ = m.Update(wheel(tui.MouseButtonWheelDown, 6, 3))
	if m.Scrolled() != 0 {
		t.Fatalf("scrolled %d", m.Scrolled())
	}
	m.Mouse = false
	m, _ = m.Update(wheel(tui.MouseButtonWheelUp, 6, 3))
	if m.Scrolled() != 0 {
		t.Fatal("Mouse off still scrolled")
	}
}

func TestMouseIsSentWhenTheProgramAsksForIt(t *testing.T) {
	m, cmd := pane(t, 30, 3, `printf '\033[?1000h\033[?1006h'; cat`)
	m.Mouse, m.Bounds = true, hittest.Rect{X: 10, Y: 4, W: 30, H: 3}
	m, cmd, _ = until(t, m, cmd, func(m Model) bool {
		m.s.mu.Lock()
		defer m.s.mu.Unlock()
		return m.s.term.Mouse() != 0
	})
	m, _ = m.Update(tui.MouseEvent{X: 12, Y: 5, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	until(t, m, cmd, has("[<0;3;2M")) // the terminal echoes the report as text
}

func TestPasteIsBracketedWhenAsked(t *testing.T) {
	m, cmd := pane(t, 40, 3, `printf '\033[?2004h'; cat`)
	m.Focus()
	m, cmd, _ = until(t, m, cmd, func(m Model) bool {
		m.s.mu.Lock()
		defer m.s.mu.Unlock()
		return m.s.term.BracketedPaste()
	})
	m, _ = m.Update(tui.PasteEvent{Text: "ab\x1b[201~cd"})
	until(t, m, cmd, has("[200~abcd"))
	if strings.Contains(text(m), "[201~ab") {
		t.Fatalf("the paste ended its own bracket:\n%s", text(m))
	}
}

func TestQueriesAreAnswered(t *testing.T) {
	m, cmd := pane(t, 40, 3, `stty raw -echo; printf '\033[6n'; dd bs=1 count=6 2>/dev/null | od -An -c`)
	until(t, m, cmd, has("1   ;   1   R"))
}

func TestTitleAndLinearize(t *testing.T) {
	m, cmd := pane(t, 30, 3, `printf '\033]0;my \033[31mtitle\007body'`)
	m, _, _ = until(t, m, cmd, has("body"))
	if got := m.Title(); got != "" && strings.ContainsAny(got, "\x1b\x07") {
		t.Fatalf("title %q", got)
	}
	m2, cmd2 := pane(t, 30, 3, `printf '\033]2;a title\007body'`)
	m2, _, _ = until(t, m2, cmd2, has("body"))
	lin := m2.Linearize()
	if m2.Title() != "a title" || !strings.HasPrefix(lin, `Terminal "a title"`) || !strings.Contains(lin, "body") {
		t.Fatalf("title %q, Linearize:\n%s", m2.Title(), lin)
	}
}

func TestCloseKillsTheProgram(t *testing.T) {
	m, _ := pane(t, 20, 3, `sleep 60`)
	m, cmd := m.Update(OutputMsg{ID: "p"}) // asks for the next message
	if !m.Running() {
		t.Fatal("not running")
	}
	if err := m.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if m.Close() != nil {
		t.Error("a second Close failed")
	}
	_, _, ex := until(t, m, cmd, func(Model) bool { return false })
	if ex == nil || ex.Code == 0 {
		t.Fatalf("the program was not killed: %+v", ex)
	}
	if m.Running() {
		t.Error("Running after Close")
	}
}

func TestStartErrorsAreReported(t *testing.T) {
	m := New("/definitely/not/a/program")
	m.ID = "x"
	msg := next(t, m.Init())
	ex, ok := msg.(ExitedMsg)
	if !ok || ex.Err == nil || ex.ID != "x" {
		t.Fatalf("msg = %#v", msg)
	}
	m = New("")
	if ex, ok := next(t, m.Init()).(ExitedMsg); !ok || ex.Err == nil {
		t.Fatal("an empty command did not fail")
	}
	// The pane still draws, blank.
	if rows := strings.Split(text(m), "\n"); len(rows) != 24 {
		t.Fatalf("%d rows", len(rows))
	}
}

func TestZeroModelIsSafe(t *testing.T) {
	var m Model
	if m.Init() != nil {
		t.Error("Init of a zero Model should do nothing")
	}
	m, cmd := m.Update(runeKey('x'))
	m.SetSize(10, 5)
	m.Input("x")
	_ = m.Close()
	_ = m.Title()
	_ = m.Running()
	if cmd != nil || strings.Count(text(m), "\n") != 4 {
		t.Fatalf("zero model: %q", text(m))
	}
	if !strings.HasPrefix(m.Linearize(), "Terminal") {
		t.Fatal(m.Linearize())
	}
}

func TestLayoutNodeCropsAndPads(t *testing.T) {
	m, cmd := pane(t, 10, 3, `printf 'abcdefghij\nxyz'`)
	m, _, _ = until(t, m, cmd, has("xyz"))
	for _, c := range []struct{ w, h int }{{10, 3}, {5, 2}, {14, 5}, {1, 1}} {
		rows := strings.Split(ansi.StripANSI(m.LayoutNode().Render(sizeOf(c.w, c.h))), "\n")
		if len(rows) != c.h {
			t.Errorf("%dx%d: %d rows", c.w, c.h, len(rows))
		}
		for _, r := range rows {
			if ansi.Width(r) != c.w {
				t.Errorf("%dx%d: a row is %d wide: %q", c.w, c.h, ansi.Width(r), r)
			}
		}
	}
	if got := m.LayoutNode().Measure(unconstrained()); got.W != 10 || got.H != 3 {
		t.Errorf("Measure = %v", got)
	}
}

func TestFocusedPaneDrawsTheCursor(t *testing.T) {
	m, cmd := pane(t, 10, 2, `printf 'ab'; sleep 5`)
	m, _, _ = until(t, m, cmd, has("ab"))
	if strings.Contains(m.View(), "\x1b[7m") {
		t.Error("an unfocused pane drew a cursor")
	}
	m.Focus()
	if !strings.Contains(m.View(), "\x1b[7m") {
		t.Errorf("a focused pane drew no cursor: %q", m.View())
	}
	m.Blur()
	if m.Focused() {
		t.Error("Blur")
	}
}

func TestHostileOutputCannotReachTheHostTerminal(t *testing.T) {
	m, cmd := pane(t, 60, 4, `printf 'a\033]52;c;SGVsbG8=\007b\033]8;;http://evil\033\\c\302\233d\342\200\256e\033[2Jf'`)
	m, _, _ = until(t, m, cmd, has("f"))
	view := m.View()
	for _, bad := range []string{"\x1b]52", "\x1b]8", "evil", "\u009b", "\u202e", "\x07"} {
		if strings.Contains(view, bad) {
			t.Errorf("the view carries %q: %q", bad, view)
		}
	}
	if !strings.Contains(m.Linearize(), "f") {
		t.Error("Linearize lost the text")
	}
}
