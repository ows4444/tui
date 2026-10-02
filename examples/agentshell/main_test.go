package main

import (
	"flag"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/toolapproval"
	"github.com/ows4444/tui/tuitest"
)

// Declared here because tuitest registers no flag: `go test -update` rewrites
// the goldens.
var _ = flag.Bool("update", false, "rewrite golden files")

var fixedNow = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// testModel is the shell with a fixed clock, no model latency and tokens that
// advance only when a test sends tokenMsg, so every frame is deterministic.
func testModel() model {
	m := initialModel()
	m.now = func() time.Time { return fixedNow }
	m.fake.latency = 0
	m.tokenEvery = 0
	return m
}

// scriptLen is how many tokenMsgs it takes to stream the whole answer and
// reach the commit that follows it.
func scriptLen() int { return len(strings.SplitAfter(scriptedAnswer, " ")) + 1 }

// newSession runs the shell inline (no alternate screen), tall enough that
// nothing scrolls out of the emulated screen.
func newSession(t *testing.T) *tuitest.Session {
	t.Helper()
	s := tuitest.New(testModel(), 70, 50, tui.WithAltScreen(false))
	t.Cleanup(s.Close)
	return s
}

func screen(s *tuitest.Session) string {
	return ansi.StripANSI(strings.Join(s.Screen(), "\n"))
}

// send submits a prompt; the model call (tui.Go) answers on its own.
func send(s *tuitest.Session, prompt string) {
	s.Keys(prompt, "enter")
}

func tokens(s *tuitest.Session, n int) {
	for i := 0; i < n; i++ {
		s.Send(tokenMsg{})
	}
}

// When the scripted stream is fed token by token, the reply grows in the live
// region; when it ends it is committed above the prompt.
func TestStreamingGolden(t *testing.T) {
	s := newSession(t)
	send(s, "run the tests")
	s.Golden(t, "sent")
	tokens(s, 8)
	s.Golden(t, "streaming")
	tokens(s, scriptLen()-8)
	s.Golden(t, "committed")
	if !strings.Contains(screen(s), "run_tests") {
		t.Fatal("tool approval did not open after the stream ended")
	}
}

// A multi-line paste that contains y, yes and Enter characters is inserted
// into the prompt and neither approves nor denies the pending tool call; only
// a real key does.
func TestPasteNeverAnswersToolCall(t *testing.T) {
	s := newSession(t)
	send(s, "run the tests")
	tokens(s, scriptLen())
	if !strings.Contains(screen(s), "run_tests") {
		t.Fatal("tool approval is not pending")
	}

	s.Paste("line one\ny\nyes\n")
	got := screen(s)
	for _, want := range []string{"run_tests", "Enter/y allow", "> line one", "yes"} {
		if !strings.Contains(got, want) {
			t.Fatalf("after paste, screen lacks %q:\n%s", want, got)
		}
	}
	for _, bad := range []string{"Denied", "Ran run_tests", "ok  example/pkg"} {
		if strings.Contains(got, bad) {
			t.Fatalf("paste resolved the tool call (%q on screen):\n%s", bad, got)
		}
	}
	s.Golden(t, "pasted")

	s.Keys("enter")
	got = screen(s)
	if !strings.Contains(got, "Ran run_tests") || strings.Contains(got, "Enter/y allow") {
		t.Fatalf("Enter did not approve:\n%s", got)
	}
	s.Golden(t, "approved")
}

// The same guarantee at the Update level: the PasteEvent yields no Cmd, leaves
// the approval pending and puts the text in the prompt.
func TestPasteEventUpdate(t *testing.T) {
	m := testModel()
	m.busy, m.pending = true, true
	m.tool = &toolCall{name: "run_tests", risk: toolapproval.RiskMedium}
	m.approve = m.newApproval()

	next, cmd := m.Update(tui.PasteEvent{Text: "line one\ny\nyes\n"})
	got := next.(model)
	if cmd != nil {
		t.Fatal("paste returned a Cmd (an approval answer would arrive that way)")
	}
	if !got.pending || got.tool == nil {
		t.Fatal("paste resolved the pending tool call")
	}
	if v := got.prompt.Value(); v != "line one\ny\nyes\n" {
		t.Fatalf("prompt = %q", v)
	}
}

// y approves, n and Esc deny; nothing is answered without a key.
func TestKeysAnswerToolCall(t *testing.T) {
	for _, tc := range []struct{ key, want string }{
		{"y", "Ran run_tests"}, {"n", "Denied run_tests"}, {"esc", "Denied run_tests"},
	} {
		s := newSession(t)
		send(s, "go")
		tokens(s, scriptLen())
		if got := screen(s); strings.Contains(got, "Ran run_tests") || strings.Contains(got, "Denied") {
			t.Fatalf("answered before any key:\n%s", got)
		}
		s.Keys(tc.key)
		if got := screen(s); !strings.Contains(got, tc.want) {
			t.Errorf("key %q: screen lacks %q:\n%s", tc.key, tc.want, got)
		}
	}
}

// In inline mode a finished message is committed with tui.Println, so it stays
// in the terminal above the live region: the user message at once, the
// assistant message when its stream ends, both above the prompt and approval.
func TestInlineScrollbackCommit(t *testing.T) {
	s := newSession(t)
	send(s, "run the tests")
	if indexOf(s.Screen(), "User") < 0 {
		t.Fatalf("user message not committed:\n%s", screen(s))
	}
	tokens(s, scriptLen())
	lines := s.Screen()
	asst, tool, prompt := indexOf(lines, "Assistant"), indexOf(lines, "run_tests"), indexOf(lines, "> ")
	if asst < 0 || prompt < 0 || tool < 0 {
		t.Fatalf("missing parts (assistant %d, tool %d, prompt %d):\n%s", asst, tool, prompt, screen(s))
	}
	if !(asst < tool && tool < prompt) {
		t.Fatalf("committed message must sit above the live region:\n%s", screen(s))
	}
	if strings.Contains(strings.Join(lines, "\n"), "thinking") {
		t.Fatalf("stale live-region text:\n%s", screen(s))
	}
}

func indexOf(lines []string, sub string) int {
	for i, l := range lines {
		if strings.Contains(ansi.StripANSI(l), sub) {
			return i
		}
	}
	return -1
}

// Submitting a prompt starts a turn: the input is cleared and a Cmd (the
// Println commit batched with the tui.Go model call) is returned.
func TestSubmitStartsTurn(t *testing.T) {
	m := testModel()
	m.prompt.SetValue("hello")
	next, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd == nil || !next.(model).busy || next.(model).prompt.Value() != "" {
		t.Fatalf("submit did not start a turn: %+v", next)
	}
}

// When a scripted session (prompt, streamed answer, tool approval) ends with
// Ctrl+C, the program exits and the final screen, which holds everything
// committed to scrollback with Println, matches the golden file.
func TestScriptedSessionExitGolden(t *testing.T) {
	s := newSession(t)
	send(s, "run the tests")
	tokens(s, scriptLen())
	s.Keys("enter")
	if s.Done() {
		t.Fatal("program exited before the session ended")
	}
	s.Keys("ctrl+c")
	if !s.Done() {
		t.Fatal("ctrl+c did not exit the program")
	}
	got := screen(s)
	for _, want := range []string{"User", "Assistant", "Ran run_tests"} {
		if !strings.Contains(got, want) {
			t.Fatalf("scrollback lacks %q after exit:\n%s", want, got)
		}
	}
	s.Golden(t, "exit")
}

func TestQuit(t *testing.T) {
	_, cmd := testModel().Update(tui.Key{Type: tui.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c should quit")
	}
}
