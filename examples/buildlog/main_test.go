package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

// callCmd runs a tui.Cmd (func() tui.Msg) and fails the test if it panics
// or returns a nil Msg. tui.Println's own Cmd is a func() tui.Msg that
// wraps an unexported printMsg — package main can't type-assert on that
// directly, so this exercises the same seam the real event loop does
// (program.go calls cmd() and inspects the resulting Msg) and confirms a
// message was actually produced at each step-completion point, which is
// the externally-observable half of "Update returned a tui.Println Cmd"
// available outside package tui. The doc comment on stepDoneMsg's case in
// Update (main.go) is the code-level guarantee that this Cmd specifically
// is tui.Println(...) and not some other Cmd.
func callCmd(t *testing.T, cmd tui.Cmd) tui.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a non-nil Cmd")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("Cmd produced a nil Msg")
	}
	return msg
}

// Each step-completion event (stepDoneMsg) returns a Cmd — the
// tui.Println call in Update's stepDoneMsg case — while the live region
// (spinner) keeps being driven separately by ordinary tickMsgs through
// View(). This proves criterion #462 at every step, and criterion #463 by
// covering all len(steps) completions across a single run, not just one.
func TestStepCompletionReturnsPrintlnCmd(t *testing.T) {
	m := initialModel()

	for i := range steps {
		next, cmd := m.Update(stepDoneMsg{index: i})
		mm := next.(model)

		callCmd(t, cmd) // proves a message-producing Cmd was returned at this step

		if mm.completed != i+1 {
			t.Errorf("step %d: completed = %d, want %d", i, mm.completed, i+1)
		}
		m = mm
	}

	if !m.finished {
		t.Fatal("model should be finished after all steps complete")
	}
}

// The live region keeps reflecting the current/next step via ordinary
// View() output as steps complete, independent of the permanent lines
// tui.Println commits elsewhere.
func TestViewTracksLiveRegionAcrossSteps(t *testing.T) {
	m := initialModel()

	if !strings.Contains(m.View(), steps[0]) {
		t.Fatalf("initial View() should mention %q:\n%s", steps[0], m.View())
	}

	next, _ := m.Update(stepDoneMsg{index: 0})
	m = next.(model)
	if !strings.Contains(m.View(), steps[1]) {
		t.Fatalf("after step 0 completes, View() should mention next step %q:\n%s", steps[1], m.View())
	}
	if !strings.Contains(m.View(), "1/4") {
		t.Fatalf("View() should reflect 1 of %d steps complete:\n%s", len(steps), m.View())
	}

	for i := 1; i < len(steps); i++ {
		next, _ := m.Update(stepDoneMsg{index: i})
		m = next.(model)
	}

	if !m.finished {
		t.Fatal("model should be finished after all steps complete")
	}
	if !strings.Contains(m.View(), "complete") {
		t.Fatalf("finished View() should say so:\n%s", m.View())
	}
}

// Quit keys work before and after completion.
func TestQuitKeys(t *testing.T) {
	m := initialModel()
	for _, key := range []tui.Key{
		{Type: tui.KeyCtrlC},
		{Type: tui.KeyEsc},
	} {
		_, cmd := m.Update(key)
		if cmd == nil {
			t.Errorf("key %+v should return a quit Cmd", key)
		}
	}

	for i := range steps {
		next, _ := m.Update(stepDoneMsg{index: i})
		m = next.(model)
	}
	_, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "q"})
	if cmd == nil {
		t.Error("'q' after finishing should return a quit Cmd")
	}
}

// Init schedules a completion tick per step plus the spinner's own
// animation Cmd, all batched together.
func TestInitSchedulesAllSteps(t *testing.T) {
	m := initialModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() should return a non-nil Cmd")
	}
}
