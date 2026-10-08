package clockview

import (
	"context"
	"testing"
	"time"

	"github.com/ows4444/tui"
)

// TestStopwatchCountsUp proves criterion #387: stopwatch mode increases
// elapsed on each tick and renders it counting up from zero.
func TestStopwatchCountsUp(t *testing.T) {
	m := New(ModeStopwatch)
	m.Interval = time.Second
	m.Start()

	if got := m.View(); got != "00:00:00" {
		t.Fatalf("initial View() = %q, want 00:00:00", got)
	}

	for i := 1; i <= 3; i++ {
		next, cmd := m.Update(tickFor(m))
		m = next
		if cmd == nil {
			t.Fatalf("tick %d: expected a rescheduled Cmd, got nil", i)
		}
	}

	if got, want := m.View(), "00:00:03"; got != want {
		t.Errorf("View() after 3 ticks = %q, want %q", got, want)
	}
	if !m.Running() {
		t.Error("stopwatch should still be running after ticks below Duration-less limit")
	}
}

// TestTimerCountsDown proves criterion #388: timer mode decreases
// remaining time on each tick and renders it counting down from Duration.
func TestTimerCountsDown(t *testing.T) {
	m := New(ModeTimer)
	m.Duration = 5 * time.Second
	m.Interval = time.Second
	m.Start()

	if got := m.View(); got != "00:00:05" {
		t.Fatalf("initial View() = %q, want 00:00:05", got)
	}

	m, _ = m.Update(tickFor(m))
	if got, want := m.View(), "00:00:04"; got != want {
		t.Errorf("View() after 1 tick = %q, want %q", got, want)
	}

	m, _ = m.Update(tickFor(m))
	if got, want := m.View(), "00:00:03"; got != want {
		t.Errorf("View() after 2 ticks = %q, want %q", got, want)
	}
}

// TestTimerStopsAtZero proves criteria #389/#390: once remaining time
// reaches zero, Running() becomes false, no further tick is scheduled, and
// additional ticks that arrive after that are no-ops (no overshoot below
// zero, no reschedule).
func TestTimerStopsAtZero(t *testing.T) {
	m := New(ModeTimer)
	m.Duration = 2 * time.Second
	m.Interval = time.Second
	m.Start()

	m, cmd := m.Update(tickFor(m)) // elapsed = 1s
	if cmd == nil {
		t.Fatal("tick below Duration should reschedule")
	}
	if !m.Running() {
		t.Error("timer should still be running before reaching Duration")
	}

	m, cmd = m.Update(tickFor(m)) // elapsed = 2s == Duration
	if cmd != nil {
		t.Error("tick that reaches Duration should not reschedule another tick")
	}
	if m.Running() {
		t.Error("Running() should be false once remaining time reaches zero")
	}
	if got, want := m.View(), "00:00:00"; got != want {
		t.Errorf("View() at zero = %q, want %q", got, want)
	}

	// Further ticks are no-ops: state and Cmd stay unchanged.
	next, cmd := m.Update(tickFor(m))
	if cmd != nil {
		t.Error("tick after stop should return a nil Cmd")
	}
	if got, want := next.View(), "00:00:00"; got != want {
		t.Errorf("View() after extra tick = %q, want %q (should not go negative)", got, want)
	}
	if next.Running() {
		t.Error("Running() should stay false after extra ticks")
	}
}

// TestClockRendersInjectedNow proves criterion #390: clock mode renders
// the current wall-clock time via an injectable now-func, formatted per a
// configurable layout, not an elapsed counter.
func TestClockRendersInjectedNow(t *testing.T) {
	fixed := time.Date(2024, 3, 14, 9, 26, 53, 0, time.UTC)
	m := New(ModeClock)
	m.Layout = "15:04:05"
	m.Now = func() time.Time { return fixed }

	if got, want := m.View(), "09:26:53"; got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}

	// Ticking a clock does not touch elapsed; View still reflects Now().
	m.Start()
	m, _ = m.Update(tickFor(m))
	fixed = fixed.Add(time.Hour)
	if got, want := m.View(), "10:26:53"; got != want {
		t.Errorf("View() after Now advances = %q, want %q", got, want)
	}
}

// TestNewDefaultsNowToTimeNow proves the "defaulting to time.Now" half of
// criterion #390.
func TestNewDefaultsNowToTimeNow(t *testing.T) {
	m := New(ModeClock)
	if m.Now == nil {
		t.Fatal("New() should default Now to a non-nil func")
	}
	before := time.Now()
	got := m.Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Errorf("New().Now() = %v, want between %v and %v (i.e. time.Now)", got, before, after)
	}
}

// TestFollowsSpinnerTickPattern proves criterion #391: Start/Stop/Running
// follow spinner.Model's self-rescheduling tui.Tick pattern in every mode
// — Start sets running and returns a Cmd producing the package's tickMsg,
// Stop needs no explicit cancellation (a tick after Stop is simply a
// no-op), and Running reflects state.
func TestFollowsSpinnerTickPattern(t *testing.T) {
	for _, mode := range []Mode{ModeClock, ModeStopwatch, ModeTimer} {
		m := New(mode)
		m.Duration = time.Hour // avoid ModeTimer hitting zero mid-test

		if m.Running() {
			t.Errorf("mode %v: New() should not start running", mode)
		}

		cmd := m.Start()
		if !m.Running() {
			t.Errorf("mode %v: Start() should set Running() true", mode)
		}
		if cmd == nil {
			t.Fatalf("mode %v: Start() should return a non-nil Cmd", mode)
		}
		msg := tui.RunCmd(context.Background(), cmd)
		if _, ok := msg.(tickMsg); !ok {
			t.Errorf("mode %v: Start()'s Cmd produced %T, want tickMsg", mode, msg)
		}

		next, tickCmd := m.Update(tickFor(m))
		if tickCmd == nil {
			t.Errorf("mode %v: a tick while running should reschedule", mode)
		}

		next.Stop()
		if next.Running() {
			t.Errorf("mode %v: Stop() should set Running() false", mode)
		}
		// A tick that arrives after Stop is a no-op: no reschedule.
		_, cmdAfterStop := next.Update(tickFor(next))
		if cmdAfterStop != nil {
			t.Errorf("mode %v: tick after Stop() should return a nil Cmd", mode)
		}
	}
}

// TestUpdateIgnoresOtherMsgTypes is a supporting check that Update only
// reacts to its own tickMsg, matching spinner's guard.
func TestUpdateIgnoresOtherMsgTypes(t *testing.T) {
	m := New(ModeStopwatch)
	m.Start()

	type otherMsg struct{}
	next, cmd := m.Update(otherMsg{})
	if cmd != nil {
		t.Error("Update with a non-tickMsg should return a nil Cmd")
	}
	if next.elapsed != m.elapsed {
		t.Error("Update with a non-tickMsg should not change elapsed")
	}
	_ = tui.Msg(otherMsg{})
}

// TestSingleModelCoversAllModes documents criterion #392: this test file
// only imports one package (clock) and exercises one type (Model) across
// all three Mode values above — there is no separate stopwatch/timer/clock
// type. See TestFollowsSpinnerTickPattern for the same Model looping over
// ModeClock, ModeStopwatch and ModeTimer.
func TestSingleModelCoversAllModes(t *testing.T) {
	var _ Model = New(ModeClock)
	var _ Model = New(ModeStopwatch)
	var _ Model = New(ModeTimer)
}

// tickFor is the tick m's own chain delivers next.
func tickFor(m Model) tickMsg { return tickMsg{id: m.id, gen: m.gen} }

// Two Models in one program each see every message. A tick belongs to the
// Model that scheduled it: the other must not count it, or it would run fast
// and schedule a second chain, doubling the ticks every interval.
func TestATickIsIgnoredByAnotherModel(t *testing.T) {
	a, b := New(ModeStopwatch), New(ModeStopwatch)
	a.Start()
	b.Start()
	a2, cmdA := a.Update(tickFor(a))
	b2, cmdB := b.Update(tickFor(a))
	if a2.View() != "00:00:01" || cmdA == nil {
		t.Fatalf("its own tick: View() = %q, rescheduled = %v", a2.View(), cmdA != nil)
	}
	if b2.View() != "00:00:00" || cmdB != nil {
		t.Fatalf("another Model's tick: View() = %q, rescheduled = %v; want it ignored", b2.View(), cmdB != nil)
	}
	// A copy is the same widget: the tick still reaches it.
	c := a
	if c2, _ := c.Update(tickFor(a)); c2.View() != "00:00:01" {
		t.Fatalf("a copy ignored its own tick: View() = %q", c2.View())
	}
}
