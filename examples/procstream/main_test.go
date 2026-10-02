package main

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

// These tests exercise Update directly with synthetic streamStartedMsg /
// streamEventMsg values rather than a real subprocess: a real child
// process's timing (scheduler jitter, sleep granularity) isn't
// deterministic enough for a unit test, but the actual message-handling
// logic under test — how Update reacts to a started process, to each
// line as it arrives, and to completion — is exactly the same whether
// the message came from a real process or was constructed by hand here.
// startProcess and waitForEvent themselves (the parts that talk to a
// real os/exec.Cmd and channel) are exercised implicitly by `go build`
// and by running the example manually.

func TestUpdate_StartProcessLaunchesWithoutBlocking(t *testing.T) {
	// Criterion #438: starting the process happens via a Cmd (returned
	// from Init), not synchronously inside Update/View. We can't observe
	// "non-blocking" from inside a unit test directly, but we can assert
	// that Init returns a Cmd (a value, not a completed action) and that
	// the resulting streamStartedMsg, once it arrives, is handled by
	// Update without requiring the process to have finished.
	m := initialModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned a nil Cmd; process launch must be deferred to a Cmd")
	}

	ch := make(chan streamEvent)
	next, updateCmd := m.Update(streamStartedMsg{events: ch})
	nm := next.(model)
	if !nm.running {
		t.Error("expected running=true after streamStartedMsg")
	}
	if updateCmd == nil {
		t.Error("expected Update to return a Cmd to wait for the next event")
	}
}

func TestUpdate_StreamsLinesIncrementally(t *testing.T) {
	// Criterion #439: each line arrives as its own Msg and is appended to
	// the log Model as it arrives, not batched at the end.
	m := initialModel()
	ch := make(chan streamEvent)
	next, _ := m.Update(streamStartedMsg{events: ch})
	m = next.(model)

	lines := []string{"line 1", "line 2", "line 3"}
	for i, line := range lines {
		next, cmd := m.Update(streamEventMsg{line: line})
		m = next.(model)

		if cmd == nil {
			t.Fatalf("after line %d, expected Update to reschedule waitForEvent", i)
		}
		if m.finished {
			t.Fatalf("after line %d, model should not be finished yet", i)
		}

		// The log should contain every line seen so far, and nothing
		// past it yet — proving lines are appended incrementally rather
		// than all at once.
		view := m.log.View()
		for j, want := range lines[:i+1] {
			if !strings.Contains(view, want) {
				t.Errorf("after line %d, log view missing %q", i, want)
			}
			_ = j
		}
		for _, notYet := range lines[i+1:] {
			if strings.Contains(view, notYet) {
				t.Errorf("after line %d, log view already contains future line %q", i, notYet)
			}
		}
	}
}

func TestUpdate_ReflectsCompletionInView(t *testing.T) {
	// Criterion #440: the View distinguishes "still running" from
	// "finished", and reflects exit status/error.
	m := initialModel()
	ch := make(chan streamEvent)
	next, _ := m.Update(streamStartedMsg{events: ch})
	m = next.(model)

	if m.finished {
		t.Fatal("model should not be finished before any streamEventMsg arrives")
	}
	if !strings.Contains(m.View(), "running") {
		t.Errorf("expected View to show a running state, got: %q", m.View())
	}

	t.Run("clean exit", func(t *testing.T) {
		next, cmd := m.Update(streamEventMsg{done: true, err: nil})
		mm := next.(model)
		if !mm.finished {
			t.Error("expected finished=true after done event")
		}
		if mm.running {
			t.Error("expected running=false after done event")
		}
		if cmd != nil {
			t.Error("expected no further Cmd once the process is done")
		}
		view := mm.View()
		if !strings.Contains(view, "finished") {
			t.Errorf("expected View to show a finished state, got: %q", view)
		}
		if strings.Contains(view, "running...") {
			t.Errorf("finished View should not still say running, got: %q", view)
		}
	})

	t.Run("process error", func(t *testing.T) {
		wantErr := errors.New("exit status 1")
		next, _ := m.Update(streamEventMsg{done: true, err: wantErr})
		mm := next.(model)
		if mm.exitErr == nil || mm.exitErr.Error() != wantErr.Error() {
			t.Errorf("expected exitErr %v, got %v", wantErr, mm.exitErr)
		}
		view := mm.View()
		if !strings.Contains(view, wantErr.Error()) {
			t.Errorf("expected View to include the error, got: %q", view)
		}
	})

	t.Run("start failure", func(t *testing.T) {
		fresh := initialModel()
		wantErr := errors.New("no such file or directory")
		next, cmd := fresh.Update(streamStartedMsg{err: wantErr})
		mm := next.(model)
		if !mm.finished {
			t.Error("a failed start should be reflected as finished, not left running forever")
		}
		if cmd != nil {
			t.Error("expected no further Cmd after a start failure")
		}
		view := mm.View()
		if !strings.Contains(view, wantErr.Error()) {
			t.Errorf("expected View to include the start error, got: %q", view)
		}
	})
}

func TestUpdate_QuitKeys(t *testing.T) {
	m := initialModel()
	for _, k := range []tui.Key{
		{Type: tui.KeyCtrlC},
		{Type: tui.KeyEsc},
		{Type: tui.KeyRunes, Text: "q"},
	} {
		_, cmd := m.Update(k)
		if cmd == nil {
			t.Errorf("expected a Cmd (Quit) for key %+v", k)
		}
	}
}

// TestDocCommentExplainsStreamingPattern guards criterion #441: main.go's
// package doc comment must explain why a single Cmd can't stream multiple
// messages and how repeated dispatch off a channel solves it. This can't
// be checked by go/doc's semantics alone, so it greps the source for the
// key claims instead of just asserting the comment's presence.
func TestDocCommentExplainsStreamingPattern(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("reading main.go: %v", err)
	}
	text := string(src)

	// Must be a package-level doc comment (directly above `package main`).
	if !strings.Contains(text, "package main") {
		t.Fatal("main.go missing `package main`")
	}
	docEnd := strings.Index(text, "package main")
	doc := text[:docEnd]
	if !strings.HasPrefix(doc, "//") {
		t.Fatal("expected a doc comment directly above `package main`")
	}

	mustContain := []string{
		"tui.Cmd", // names the type being discussed
		"once",    // a Cmd only ever produces one Msg
		"channel", // the fix: read repeatedly off a channel
	}
	for _, want := range mustContain {
		if !strings.Contains(doc, want) {
			t.Errorf("package doc comment missing expected term %q", want)
		}
	}
}
