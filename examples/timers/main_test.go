package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/tuitest"
)

// nine is a fixed source of time for the wall clock.
func nine() time.Time { return time.Date(2026, time.August, 14, 9, 0, 0, 0, time.UTC) }

// The two tests that run the program wait on its real ticks, a second each,
// so they run in parallel and allow a slow machine plenty of time.
const patience = 20 * time.Second

func start(t *testing.T) *tuitest.Session {
	t.Helper()
	s := tuitest.New(newModel(nine), 60, 12)
	t.Cleanup(s.Close)
	return s
}

func waitFor(t *testing.T, s *tuitest.Session, want string) {
	t.Helper()
	if !s.WaitForText(want, patience) {
		t.Fatalf("the screen never showed %q:\n%s", want, strings.Join(s.Screen(), "\n"))
	}
}

func TestCountsDownThenShowsTheReport(t *testing.T) {
	t.Parallel()
	s := start(t)
	waitFor(t, s, "09:00:00")
	waitFor(t, s, "Loading, 00:00:02 left")
	if strings.Contains(strings.Join(s.Screen(), "\n"), report[0]) {
		t.Fatal("the report is shown before it has loaded")
	}
	// The stopwatch starts when the report arrives and counts its age.
	waitFor(t, s, "Loaded 00:00:00 ago")
	waitFor(t, s, report[0])
	waitFor(t, s, report[2])
	waitFor(t, s, "Loaded 00:00:01 ago")
}

// Loading again partway through leaves ticks from the first load on their
// way. They belong to a widget that was replaced, so the new wait is a full
// one.
func TestLoadingAgainMidwayStartsAFullWait(t *testing.T) {
	t.Parallel()
	s := start(t)
	waitFor(t, s, "Loading, 00:00:02 left")
	s.Keys("r")
	waitFor(t, s, "Loading, 00:00:03 left")
	waitFor(t, s, "Loading, 00:00:02 left")
	waitFor(t, s, "Loading, 00:00:01 left")
	waitFor(t, s, "Loaded 00:00:00 ago")
}

// The rest drive the model directly, with no ticks.

func press(m model, k tui.Key) (model, tui.Cmd) {
	next, cmd := m.Update(k)
	return next.(model), cmd
}

var (
	space = tui.Key{Type: tui.KeySpace, Text: " ", Code: ' '}
	keyR  = tui.Key{Type: tui.KeyRunes, Text: "r", Code: 'r'}
	keyQ  = tui.Key{Type: tui.KeyRunes, Text: "q", Code: 'q'}
)

func TestSpacePausesAndResumesTheWait(t *testing.T) {
	m, cmd := press(newModel(nine), space)
	if !m.paused || m.wait.Running() || m.ghost.Running() || cmd != nil {
		t.Fatalf("after space: paused=%v, countdown running=%v, skeleton running=%v, Cmd=%v",
			m.paused, m.wait.Running(), m.ghost.Running(), cmd != nil)
	}
	if !m.clock.Running() {
		t.Fatal("pausing the wait stopped the wall clock")
	}
	out := ansi.StripANSI(m.View())
	if !strings.Contains(out, "Paused with 00:00:03 left") || !strings.Contains(out, "resume") {
		t.Fatalf("the paused screen:\n%s", out)
	}
	// A message that arrives while paused does not count as the report
	// arriving, although the countdown is not running.
	next, _ := m.Update(struct{}{})
	if m = next.(model); !m.loading {
		t.Fatal("a message during the pause ended the load")
	}
	m, cmd = press(m, space)
	if m.paused || !m.wait.Running() || !m.ghost.Running() || cmd == nil {
		t.Fatalf("after a second space: paused=%v, countdown running=%v, skeleton running=%v, Cmd=%v",
			m.paused, m.wait.Running(), m.ghost.Running(), cmd != nil)
	}
}

func TestTheReportReplacesTheSkeletonWhenTheCountdownStops(t *testing.T) {
	m := newModel(nine)
	if out := ansi.StripANSI(m.View()); strings.Contains(out, report[0]) {
		t.Fatalf("the report is shown while loading:\n%s", out)
	}
	m.wait.Stop() // as the countdown does to itself at zero
	next, cmd := m.Update(struct{}{})
	m = next.(model)
	if m.loading || m.ghost.Running() || !m.age.Running() || cmd == nil {
		t.Fatalf("loading=%v, skeleton running=%v, stopwatch running=%v, Cmd=%v; want the load over and the stopwatch started",
			m.loading, m.ghost.Running(), m.age.Running(), cmd != nil)
	}
	out := ansi.StripANSI(m.View())
	if !strings.Contains(out, "Loaded 00:00:00 ago") || !strings.Contains(out, report[1]) || strings.Contains(out, "space") {
		t.Fatalf("the loaded screen:\n%s", out)
	}
	// Space does nothing once the report is shown.
	if m, cmd := press(m, space); m.paused || cmd != nil {
		t.Fatal("space paused a load that had finished")
	}
}

func TestRStartsANewLoad(t *testing.T) {
	m := newModel(nine)
	m.wait.Stop()
	next, _ := m.Update(struct{}{})
	m, cmd := press(next.(model), keyR)
	if !m.loading || !m.wait.Running() || m.age.Running() || !m.ghost.Running() || cmd == nil {
		t.Fatalf("after r: loading=%v, countdown running=%v, stopwatch running=%v, skeleton running=%v, Cmd=%v",
			m.loading, m.wait.Running(), m.age.Running(), m.ghost.Running(), cmd != nil)
	}
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "Loading, 00:00:03 left") {
		t.Fatalf("the screen after r:\n%s", out)
	}
}

func TestQuitKeys(t *testing.T) {
	for name, k := range map[string]tui.Key{"q": keyQ, "ctrl+c": {Type: tui.KeyCtrlC}} {
		_, cmd := press(newModel(nine), k)
		if cmd == nil {
			t.Fatalf("%s returned no Cmd", name)
		}
		if _, ok := cmd().(tui.QuitMsg); !ok {
			t.Errorf("%s did not quit", name)
		}
	}
}
