package main

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

// Tests drive Update with a synthetic loadedMsg directly instead of
// waiting out the real fetchCmd's time.Sleep, so they stay fast and
// deterministic rather than depending on real-time timing.

// TestInitDispatchesLoadAndShowsSpinner covers criterion #442: on
// startup, Init dispatches a Cmd simulating the slow operation and the
// model starts in the loading state with a spinner running.
func TestInitDispatchesLoadAndShowsSpinner(t *testing.T) {
	m := initialModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() should return a non-nil Cmd")
	}

	// initialModel already starts in the loading state (Init only adds
	// the Cmds on top of it, since Init can't return a mutated Model),
	// so the first View() painted matches what main() actually renders.
	if !strings.Contains(m.View(), "Loading...") {
		t.Errorf("initial View() should show the loading state, got:\n%s", m.View())
	}
}

// TestLoadedMsgSwapsViewAndStopsSpinner covers criterion #443: when the
// simulated operation's Cmd resolves (delivered here as a synthetic
// loadedMsg, bypassing the real timer), the View swaps from the loading
// indicator to the resolved content and the spinner is explicitly stopped.
func TestLoadedMsgSwapsViewAndStopsSpinner(t *testing.T) {
	m := initialModel()
	m.loading = true
	m.spinner.Start()

	if !m.spinner.Running() {
		t.Fatal("setup: spinner should be running before resolution")
	}

	next, cmd := m.Update(loadedMsg{result: "test result"})
	m2 := next.(model)

	if m2.loading {
		t.Error("loading should be false once loadedMsg is delivered")
	}
	if m2.spinner.Running() {
		t.Error("spinner should be explicitly stopped once loadedMsg is delivered")
	}
	if cmd != nil {
		t.Error("resolving loadedMsg should not schedule another Cmd")
	}

	view := m2.View()
	if strings.Contains(view, "Loading...") {
		t.Errorf("View() should no longer show the loading indicator, got:\n%s", view)
	}
	if !strings.Contains(view, "test result") {
		t.Errorf("View() should show the resolved content, got:\n%s", view)
	}
}

// TestSpinnerTicksWhileLoading exercises the in-flight window: while
// still loading, unrelated Msgs (here the spinner's own tick, routed
// through Update's default branch) keep the spinner animating rather than
// being dropped, matching the "UI stays responsive while the Cmd is in
// flight" behavior the doc comment describes.
func TestSpinnerTicksWhileLoading(t *testing.T) {
	m := initialModel()
	m.loading = true
	startCmd := m.spinner.Start()
	if startCmd == nil {
		t.Fatal("spinner.Start() should return a tick Cmd")
	}
	tickMsg := tui.RunCmd(context.Background(), startCmd)

	next, cmd := m.Update(tickMsg)
	m2 := next.(model)

	if !m2.loading {
		t.Fatal("model should still be loading after an unrelated tick")
	}
	if cmd == nil {
		t.Error("spinner should reschedule its own tick while loading")
	}
}

// TestDocCommentExplainsAsyncPattern covers criterion #444: main.go has a
// doc comment explaining the async-Cmd-resolves-into-a-Msg pattern.
func TestDocCommentExplainsAsyncPattern(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("reading main.go: %v", err)
	}
	src := string(b)
	for _, want := range []string{"Cmd", "Msg", "goroutine"} {
		if !strings.Contains(src, want) {
			t.Errorf("main.go doc comment should mention %q", want)
		}
	}
}

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
}
