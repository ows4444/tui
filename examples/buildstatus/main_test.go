package main

import (
	"flag"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/tuitest"
)

var _ = flag.Bool("update", false, "rewrite golden files")

// scripted is the model without its first timer, so the test alone decides when
// each tick arrives and no wall-clock time is involved.
type scripted struct{ model }

func (scripted) Init() tui.Cmd { return nil }

func (s scripted) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	m, cmd := s.model.Update(msg)
	return scripted{m.(model)}, cmd
}

func session(t *testing.T, jobs int, fail string) *tuitest.Session {
	t.Helper()
	m := newModel(jobs, fail)
	m.still, m.manual = false, true
	s := tuitest.New(scripted{m}, 100, 40, tui.WithAltScreen(false))
	t.Cleanup(s.Close)
	return s
}

func screen(s *tuitest.Session) string { return strings.Join(s.Screen(), "\n") }

func finish(s *tuitest.Session) {
	for n := 0; !s.Done() && n < 500; n++ {
		s.Send(tickMsg{})
	}
}

func TestSuccessfulBuildLeavesOneLinePerStepAndASummary(t *testing.T) {
	s := session(t, 3, "")
	finish(s)
	if !s.Done() {
		t.Fatal("the program did not end")
	}
	out := screen(s)
	for _, st := range pipeline {
		if !strings.Contains(out, "✓ "+st.name) {
			t.Errorf("scrollback lacks %q:\n%s", st.name, out)
		}
	}
	if !strings.Contains(out, "Build succeeded in") || !strings.Contains(out, "8 steps") || strings.Contains(out, "✗") {
		t.Fatalf("\n%s", out)
	}
	if strings.Index(out, "✓ fetch modules") > strings.Index(out, "✓ test") {
		t.Fatal("steps were printed out of order")
	}
}

func TestFailedBuildPrintsTheOutputAndWhatWasSkipped(t *testing.T) {
	s := session(t, 3, "test")
	finish(s)
	out := screen(s)
	for _, want := range []string{"✗ test", "failed", "│ --- FAIL: TestWrap", "wrap_test.go:41: got 3 rows, want 2", "- package", "skipped: test did not finish", "Build failed after", "1 failed, 1 skipped, 6 ok"} {
		if !strings.Contains(out, want) {
			t.Errorf("scrollback lacks %q:\n%s", want, out)
		}
	}
}

func TestCtrlCCancelsAndKeepsWhatFinished(t *testing.T) {
	s := session(t, 3, "")
	for range 30 {
		s.Send(tickMsg{})
	}
	s.Keys("ctrl+c")
	if !s.Done() {
		t.Fatal("did not exit")
	}
	out := screen(s)
	for _, want := range []string{"✓ fetch modules", "✓ compile ansi", "Build cancelled after 3.0s", "- test", "cancelled"} {
		if !strings.Contains(out, want) {
			t.Errorf("lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "✓ test") {
		t.Fatal("a stopped step was reported finished")
	}
}

func TestLiveRegionShowsRunningAndWaitingStepsOnly(t *testing.T) {
	m := newModel(3, "")
	m.still = false
	m.width = 100
	var ended []int
	for range 12 {
		m.b, ended = m.b.advance()
	}
	_ = ended
	out := ansi.StripANSI(m.View())
	for _, want := range []string{"Building  1.2s", "1/8 done", "3 running, up to 3", "compile ansi", "compile widgets", "% ", "· link tui", "waits for compile ansi, compile layout, compile widgets", "ctrl+c cancel"} {
		if !strings.Contains(out, want) {
			t.Errorf("live region lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "fetch modules") {
		t.Fatalf("a finished step is still in the live region:\n%s", out)
	}
}

func TestVerboseKeyHidesOutputLines(t *testing.T) {
	m := newModel(3, "")
	m.width = 100
	for range 12 {
		m.b, _ = m.b.advance()
	}
	with := ansi.StripANSI(m.View())
	next, _ := m.Update(tui.Key{Type: tui.KeyRunes, Text: "v"})
	without := ansi.StripANSI(next.(model).View())
	if !strings.Contains(with, "compile layout/box.go") && !strings.Contains(with, "compile ansi/color.go") {
		t.Fatalf("no output line:\n%s", with)
	}
	if strings.Contains(without, "compile ansi/") || strings.Contains(without, "compile layout/") {
		t.Fatalf("output lines still shown:\n%s", without)
	}
}

func TestPlusAndMinusChangeTheLimit(t *testing.T) {
	m := newModel(3, "")
	next, _ := m.Update(tui.Key{Type: tui.KeyRunes, Text: "-"})
	m = next.(model)
	next, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "-"})
	next, _ = next.Update(tui.Key{Type: tui.KeyRunes, Text: "-"})
	if next.(model).b.max != 1 {
		t.Fatalf("max %d", next.(model).b.max)
	}
	next, _ = next.Update(tui.Key{Type: tui.KeyRunes, Text: "+"})
	for range 20 {
		next, _ = next.Update(tui.Key{Type: tui.KeyRunes, Text: "+"})
	}
	if next.(model).b.max != len(pipeline) {
		t.Fatalf("max %d", next.(model).b.max)
	}
}

func TestReducedMotionDrawsNoSpinnerFrames(t *testing.T) {
	m := newModel(3, "")
	m.still, m.width = true, 100
	for range 12 {
		m.b, _ = m.b.advance()
	}
	out := ansi.StripANSI(m.View())
	if !strings.Contains(out, "… compile") {
		t.Fatalf("\n%s", out)
	}
	for _, f := range spinnerFrames() {
		if strings.Contains(out, f) {
			t.Fatalf("spinner frame %q drawn under reduced motion", f)
		}
	}
}

func spinnerFrames() []string { return newModel(1, "").frames }

func TestHelpAndQuit(t *testing.T) {
	m := newModel(3, "")
	next, _ := m.Update(tui.Key{Type: tui.KeyRunes, Text: "?"})
	if !strings.Contains(ansi.StripANSI(next.View()), "cancel the build") {
		t.Fatal("no help")
	}
	next, cmd := next.Update(tui.Key{Type: tui.KeyEsc})
	if next.(model).help || cmd != nil {
		t.Fatal("Esc should only close help")
	}
	if _, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "q"}); cmd != nil {
		t.Fatal("q must not quit a build in progress")
	}
	done := m
	done.b = done.b.cancel()
	if _, cmd := done.Update(tui.Key{Type: tui.KeyRunes, Text: "q"}); cmd == nil {
		t.Fatal("q should leave a finished build")
	}
	if _, cmd := done.Update(tui.Key{Type: tui.KeyEnter}); cmd == nil {
		t.Fatal("Enter should leave a finished build")
	}
}
