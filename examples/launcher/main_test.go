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

func typed(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready() model {
	return send(newModel(sampleCommands(), false, 0), tui.ResizeMsg{Width: 80, Height: 24})
}

// drain feeds the model every message cmd and its successors produce, like the
// Program does, until nothing is left to read.
func drain(m model, cmd tui.Cmd) model {
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			return m
		}
		m, cmd = update(m, msg)
	}
	return m
}

// enter presses Enter and lets the run finish.
func enter(m model) model {
	m, cmd := update(m, tui.Key{Type: tui.KeyEnter})
	return drain(m, cmd)
}

func text(m model) string { return ansi.StripANSI(m.View()) }

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no target %q on screen:\n%s", id, text(m))
	}
	return r
}

func press(m model, x, y, clicks int) (model, tui.Cmd) {
	return update(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: clicks})
}

func names(m model) []string {
	var out []string
	for _, e := range m.matches() {
		out = append(out, m.cmds[e.idx].name)
	}
	return out
}

func TestStartsWithTheCommandsAndNoOutput(t *testing.T) {
	out := text(ready())
	for _, want := range []string{"launcher", "sample project, scripted", "Type a command name", "build", "Compile the project", "Output appears here"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestTypingNarrowsAndRanks(t *testing.T) {
	m := typed(ready(), "dp")
	got := names(m)
	if len(got) == 0 || got[0] != "deploy-production" && got[0] != "deploy-staging" {
		t.Fatalf("matches for dp = %v", got)
	}
	m = key(m, tui.KeyEsc)
	if m.in.Value() != "" || len(names(m)) != len(sampleCommands()) {
		t.Fatal("Esc did not clear the filter")
	}
	m = typed(m, "zzzzzq")
	if !strings.Contains(text(m), "No command matches.") {
		t.Fatalf("\n%s", text(m))
	}
	if _, cmd := update(m, tui.Key{Type: tui.KeyEnter}); cmd != nil {
		t.Fatal("Enter with no match ran something")
	}
}

func TestFilterSearchesDescriptions(t *testing.T) {
	m := typed(ready(), "zq")
	if len(names(m)) != 0 {
		t.Fatalf("zq matched %v", names(m))
	}
	m = key(m, tui.KeyEsc)
	m = typed(m, "benchmarks")
	if got := names(m); len(got) == 0 || got[0] != "bench" {
		t.Fatalf("matches for benchmarks = %v", got)
	}
}

func TestEnterRunsAndStreamsOutput(t *testing.T) {
	m := typed(ready(), "fmt")
	m, cmd := update(m, tui.Key{Type: tui.KeyEnter})
	if cmd == nil || !m.last.running || m.last.name != "fmt" {
		t.Fatalf("run did not start: %+v", m.last)
	}
	if !strings.Contains(text(m), "▶ running") {
		t.Fatalf("no running state:\n%s", text(m))
	}
	m = drain(m, cmd)
	out := text(m)
	for _, want := range []string{"gofmt -l -w .", "3 files changed", "✓ exit 0", "[Rerun]"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if m.in.Value() != "" {
		t.Error("the filter was kept after running")
	}
}

func TestFailureShowsTheExitCode(t *testing.T) {
	m := typed(ready(), "deploy-production")
	m = enter(m)
	out := text(m)
	if !strings.Contains(out, "✗ exit 2") || !strings.Contains(out, "no sign-off") {
		t.Fatalf("\n%s", out)
	}
}

func TestRecentCommandsComeFirstAndAreMarked(t *testing.T) {
	m := ready()
	for _, n := range []string{"lint", "docs", "lint"} {
		m = typed(m, n)
		m = enter(m)
	}
	got := names(m)
	if got[0] != "lint" || got[1] != "docs" {
		t.Fatalf("order = %v, want lint then docs first", got[:3])
	}
	if len(m.recents) != 2 {
		t.Fatalf("recents = %v", m.recents)
	}
	if !strings.Contains(text(m), "↺ lint") {
		t.Fatalf("no recent mark:\n%s", text(m))
	}
}

func TestRecentsAreBounded(t *testing.T) {
	m := ready()
	for _, n := range []string{"build", "test", "lint", "fmt", "bench", "docs", "clean"} {
		m = enter(typed(m, n))
	}
	if len(m.recents) != maxRecents || m.recents[0] != "clean" {
		t.Fatalf("recents = %v", m.recents)
	}
}

func TestRecencyBreaksTiesInRanking(t *testing.T) {
	m := ready()
	m = enter(typed(m, "deploy-production"))
	m = typed(m, "deploy")
	if got := names(m); got[0] != "deploy-production" {
		t.Fatalf("an equal match did not rank the recent one first: %v", got)
	}
}

func TestRerunAndCtrlR(t *testing.T) {
	m := enter(typed(ready(), "fmt"))
	before := m.seq
	r := rect(t, m, "rerun")
	m, cmd := press(m, r.X+1, r.Y, 1)
	m = drain(m, cmd)
	if m.seq != before+1 || len(m.last.lines) != 2 || m.last.running {
		t.Fatalf("Rerun: seq=%d lines=%d", m.seq, len(m.last.lines))
	}
	m, cmd = update(m, tui.Key{Type: tui.KeyCtrl, Code: 'r'})
	m = drain(m, cmd)
	if m.seq != before+2 {
		t.Fatalf("Ctrl+R did not run again: seq=%d", m.seq)
	}
}

func TestCancelStopsARun(t *testing.T) {
	m := newModel(sampleCommands(), false, 50e6)
	m = send(m, tui.ResizeMsg{Width: 80, Height: 24})
	m = typed(m, "release")
	m, cmd := update(m, tui.Key{Type: tui.KeyEnter})
	m = drain(m, nil)
	r := rect(t, m, "cancel")
	m, _ = press(m, r.X+1, r.Y, 1)
	m = drain(m, cmd)
	if m.last.running || !strings.Contains(text(m), "✗ cancelled") {
		t.Fatalf("not cancelled:\n%s", text(m))
	}
	if len(m.last.lines) >= 5 {
		t.Fatalf("the whole script printed (%d lines)", len(m.last.lines))
	}
}

func TestCtrlXCancels(t *testing.T) {
	m := newModel(sampleCommands(), false, 50e6)
	m = send(m, tui.ResizeMsg{Width: 80, Height: 24})
	m, cmd := update(typed(m, "release"), tui.Key{Type: tui.KeyEnter})
	m = send(m, tui.Key{Type: tui.KeyCtrl, Code: 'x'})
	m = drain(m, cmd)
	if m.last.running || m.last.err == nil {
		t.Fatalf("running=%v err=%v", m.last.running, m.last.err)
	}
}

func TestNewRunReplacesTheOldAndIgnoresItsLines(t *testing.T) {
	m := newModel(sampleCommands(), false, 30e6)
	m = send(m, tui.ResizeMsg{Width: 80, Height: 24})
	m, first := update(typed(m, "release"), tui.Key{Type: tui.KeyEnter})
	oldSeq := m.seq
	m, second := update(typed(m, "fmt"), tui.Key{Type: tui.KeyEnter})
	_ = first
	m = send(m, outMsg{seq: oldSeq, line: "stale line"})
	m = drain(m, second)
	if m.last.name != "fmt" || strings.Contains(strings.Join(m.last.lines, "\n"), "stale") {
		t.Fatalf("name=%s lines=%v", m.last.name, m.last.lines)
	}
}

func TestOutputIsBoundedAndScrollKeepsThePlace(t *testing.T) {
	m := ready()
	m.last = run{name: "x", seq: 1}
	m.seq = 1
	for i := 0; i < maxLines+50; i++ {
		m, _ = update(m, outMsg{seq: 1, line: "l"})
	}
	if len(m.last.lines) != maxLines {
		t.Fatalf("%d lines kept, want %d", len(m.last.lines), maxLines)
	}
	m = key(m, tui.KeyPgUp)
	at := m.scroll
	if at == 0 {
		t.Fatal("PgUp did not scroll")
	}
	m, _ = update(m, outMsg{seq: 1, line: "new"})
	if m.scroll != at+1 {
		t.Fatalf("scroll %d -> %d: the place being read moved", at, m.scroll)
	}
	if !strings.Contains(text(m), "newer lines") {
		t.Fatalf("no marker for newer lines:\n%s", text(m))
	}
	m = key(m, tui.KeyPgDown)
	for i := 0; i < 2000; i++ {
		m = key(m, tui.KeyPgDown)
	}
	if m.scroll != 0 {
		t.Fatalf("scroll = %d at the bottom", m.scroll)
	}
}

func TestClickSelectsAndDoubleClickRuns(t *testing.T) {
	m := ready()
	r := rect(t, m, "row-2")
	m, cmd := press(m, r.X+4, r.Y, 1)
	if m.cur != 2 || cmd != nil {
		t.Fatalf("cur=%d cmd=%v after a click", m.cur, cmd != nil)
	}
	m, cmd = press(m, r.X+4, r.Y, 2)
	m = drain(m, cmd)
	if m.last.name != names(newModel(sampleCommands(), false, 0))[2] && m.last.name == "" {
		t.Fatalf("double click ran %q", m.last.name)
	}
	if m.last.name != "test-flaky" {
		t.Fatalf("double click on the third row ran %q", m.last.name)
	}
}

func TestWheelOverListAndOutput(t *testing.T) {
	m := send(ready(), tui.ResizeMsg{Width: 80, Height: 12}) // too short to show the whole output
	m = enter(typed(m, "test"))
	lr := rect(t, m, "row-0")
	m = send(m, tui.MouseEvent{X: lr.X, Y: lr.Y, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.cur != 1 {
		t.Fatalf("cur = %d after the wheel over the list", m.cur)
	}
	or := rect(t, m, "out")
	m = send(m, tui.MouseEvent{X: or.X, Y: or.Y, Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.scroll == 0 {
		t.Fatal("the wheel over the output did not scroll it")
	}
}

func TestListFollowsTheSelection(t *testing.T) {
	m := send(ready(), tui.ResizeMsg{Width: 80, Height: 14})
	for i := 0; i < 9; i++ {
		m = key(m, tui.KeyDown)
	}
	sel, _ := m.selected()
	if !strings.Contains(text(m), sel.name) {
		t.Fatalf("the selected command %s is not on screen:\n%s", sel.name, text(m))
	}
	if m = send(m, tui.ResizeMsg{Width: 80, Height: 14}); m.top() == 0 {
		t.Fatalf("top = 0 with the selection at %d and %d rows", m.cur, m.listRows())
	}
}

func TestEscClearsTheFilterAndCtrlCQuits(t *testing.T) {
	m := typed(ready(), "te")
	if m = key(m, tui.KeyEsc); m.cur != 0 || m.in.Value() != "" {
		t.Fatal("Esc did not reset")
	}
	if _, cmd := update(ready(), tui.Key{Type: tui.KeyCtrlC}); !isQuit(cmd) {
		t.Fatal("Ctrl+C did not quit")
	}
	if _, cmd := update(ready(), tui.Key{Type: tui.KeyCtrl, Code: 'q'}); !isQuit(cmd) {
		t.Fatal("Ctrl+Q did not quit")
	}
	if _, cmd := update(typed(ready(), "q"), tui.Key{Type: tui.KeyRunes, Text: "q"}); isQuit(cmd) {
		t.Fatal("typing q quit")
	}
}

func TestQuitCancelsARunningCommand(t *testing.T) {
	m := newModel(sampleCommands(), false, 50e6)
	m = send(m, tui.ResizeMsg{Width: 80, Height: 24})
	m, cmd := update(typed(m, "release"), tui.Key{Type: tui.KeyEnter})
	if _, q := update(m, tui.Key{Type: tui.KeyCtrlC}); !isQuit(q) {
		t.Fatal("Ctrl+C did not quit")
	}
	m = drain(m, cmd)
	if m.last.running {
		t.Fatal("the run kept going after quit")
	}
}

func TestHelp(t *testing.T) {
	m := key(ready(), tui.KeyF1)
	if !strings.Contains(text(m), "run the last one again") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
}

func TestShellCommandsRunThroughTheModel(t *testing.T) {
	skipWithoutShell(t)
	cs, _ := parseCommands("hello | Say hello | echo hello from sh\nfail | Fail | echo oops; exit 5\n")
	m := send(newModel(cs, true, 0), tui.ResizeMsg{Width: 80, Height: 24})
	if !strings.Contains(text(m), "your commands, run for real") {
		t.Fatalf("\n%s", text(m))
	}
	m = enter(typed(m, "hello"))
	if !strings.Contains(text(m), "hello from sh") || !strings.Contains(text(m), "✓ exit 0") {
		t.Fatalf("\n%s", text(m))
	}
	m = enter(typed(m, "fail"))
	if !strings.Contains(text(m), "oops") || !strings.Contains(text(m), "✗ exit 5") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m := enter(typed(ready(), "test"))
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}
