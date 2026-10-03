package motion

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestClockAdvancesOnItsOwnTickAndSchedulesNext(t *testing.T) {
	c := NewClock(time.Millisecond)
	cmd := c.Start()
	if cmd == nil {
		t.Fatal("Start returned nil")
	}
	if c.Start() != nil {
		t.Error("second Start must not start another chain")
	}
	msg := cmd(context.Background())
	if next := c.Update(msg); next == nil || c.Frame() != 1 {
		t.Fatalf("frame=%d next=%v", c.Frame(), next != nil)
	}
	if c.Update(TickMsg{Clock: NewClock(time.Millisecond)}) != nil || c.Frame() != 1 {
		t.Error("another clock's tick must be ignored")
	}
	c.Stop()
	if c.Update(msg) != nil || c.Frame() != 1 {
		t.Error("tick after Stop must be a no-op")
	}
}

// TestSharedClockDeliversOneTickPerInterval proves criterion #84: however
// many widgets Start the same Clock, only one tick chain exists, so each
// interval yields exactly one TickMsg and one Frame step.
func TestSharedClockDeliversOneTickPerInterval(t *testing.T) {
	c := NewClock(time.Millisecond)
	var cmds []func(context.Context) any
	for i := 0; i < 5; i++ { // five widgets each ask to start the clock
		if cmd := c.Start(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) != 1 {
		t.Fatalf("%d tick Cmds from 5 Starts, want 1", len(cmds))
	}
	// Follow the chain: each Update yields exactly one next Cmd and one
	// frame.
	cmd := cmds[0]
	for want := 1; want <= 3; want++ {
		msg := cmd(context.Background())
		if tm, ok := msg.(TickMsg); !ok || tm.Clock != c {
			t.Fatalf("msg = %#v", msg)
		}
		cmd = c.Update(msg)
		if cmd == nil || c.Frame() != want {
			t.Fatalf("frame=%d want %d, next=%v", c.Frame(), want, cmd != nil)
		}
	}
}

func TestClockReducedSchedulesNothing(t *testing.T) {
	c := &Clock{Interval: time.Millisecond, Motion: Reduced}
	if c.Start() != nil || c.Running() {
		t.Error("reduced Start scheduled a tick")
	}
	if c.Update(TickMsg{Clock: c}) != nil || c.Frame() != 0 {
		t.Error("reduced Update advanced or scheduled")
	}
	var nilc *Clock
	if nilc.Start() != nil || nilc.Frame() != 0 || nilc.Running() {
		t.Error("nil Clock must be inert")
	}
}

// TestClockDriftFree proves criterion #33: with a 5 ms Update cost a 16 ms
// Clock still delivers one tick per slot, 60-64 over one second. It counts
// wall-clock slots, which a loaded machine misses, so like the other timing
// tests it runs only when TUI_TIMING_TESTS is set.
func TestClockDriftFree(t *testing.T) {
	if os.Getenv("TUI_TIMING_TESTS") == "" {
		t.Skip("wall-clock test; set TUI_TIMING_TESTS=1 to run it")
	}
	c := NewClock(16 * time.Millisecond)
	cmd := c.Start()
	deadline := time.Now().Add(time.Second)
	ticks := 0
	for cmd != nil && time.Now().Before(deadline) {
		msg := cmd(context.Background())
		if time.Now().After(deadline) {
			break
		}
		ticks++
		time.Sleep(5 * time.Millisecond) // simulated Update cost
		cmd = c.Update(msg)
	}
	if ticks < 60 || ticks > 64 {
		t.Fatalf("ticks = %d, want 60-64", ticks)
	}
}

func TestClockRestartDropsStaleTick(t *testing.T) {
	c := NewClock(time.Millisecond)
	old := c.Start()
	c.Stop()
	fresh := c.Start()
	if c.Update(old(context.Background())) != nil || c.Frame() != 0 {
		t.Error("tick from before the restart must be dropped")
	}
	if c.Update(fresh(context.Background())) == nil || c.Frame() != 1 {
		t.Error("tick from the current run must advance")
	}
}

// TestFrameClockIdleSchedulesNothing proves criterion #34: with no
// animator the Clock has no timer, and a stale tick is absorbed.
func TestFrameClockIdleSchedulesNothing(t *testing.T) {
	c := NewFrameClock(60)
	if c.Running() {
		t.Fatal("new Clock is running")
	}
	cmd := c.Acquire()
	msg := cmd(context.Background())
	c.Release()
	if c.Running() || c.Update(msg) != nil {
		t.Error("Clock kept a timer after the last Release")
	}
	c.Release() // extra Release is harmless
	if c.Acquire() == nil {
		t.Error("Clock did not restart on a new Acquire")
	}
}

// TestFrameClockOneTickPerFrame proves criterion #35: several animators
// share one chain, so each frame interval yields one tick.
func TestFrameClockOneTickPerFrame(t *testing.T) {
	c := NewFrameClock(60)
	if got := c.Interval; got != time.Second/60 {
		t.Fatalf("Interval = %v", got)
	}
	var cmds []func(context.Context) any
	for i := 0; i < 12; i++ {
		if cmd := c.Acquire(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) != 1 {
		t.Fatalf("%d tick chains from 12 animators, want 1", len(cmds))
	}
	cmd, start := cmds[0], time.Now()
	for i := 0; i < 5; i++ {
		cmd = c.Update(cmd(context.Background()))
		if cmd == nil {
			t.Fatal("chain ended")
		}
	}
	if el := time.Since(start); el < 4*c.Interval {
		t.Errorf("5 ticks in %v: faster than one per %v", el, c.Interval)
	}
	if FrameInterval(0) != time.Second/DefaultFPS || FrameInterval(30) != time.Second/30 {
		t.Error("FrameInterval")
	}
}

// manualSource is a Source whose tickers fire only when the test sends on them.
type manualSource struct {
	now     time.Time
	ticks   chan time.Time
	periods []time.Duration
	stopped int
}

func (s *manualSource) Now() time.Time { return s.now }

func (s *manualSource) NewTicker(d time.Duration) (<-chan time.Time, func()) {
	s.periods = append(s.periods, d)
	return s.ticks, func() { s.stopped++ }
}

func TestAfterOnWaitsForTheSourceTick(t *testing.T) {
	src := &manualSource{now: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), ticks: make(chan time.Time)}
	at := src.now.Add(time.Hour)
	got := make(chan time.Time, 1)
	go func() {
		got <- AfterOn(&Clock{Source: src}, time.Hour, func(t time.Time) time.Time { return t })(context.Background())
	}()
	src.ticks <- at // a wall-clock hour never passes: only this tick ends the wait
	if g := <-got; !g.Equal(at) {
		t.Fatalf("fn got %v, want the tick's time %v", g, at)
	}
	if len(src.periods) != 1 || src.periods[0] != time.Hour || src.stopped != 1 {
		t.Fatalf("ticker periods %v, stopped %d; want one ticker of 1h, stopped once", src.periods, src.stopped)
	}
}

func TestAfterOnCancelledSkipsFnAndStopsTheTicker(t *testing.T) {
	src := &manualSource{ticks: make(chan time.Time)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	got := AfterOn(&Clock{Source: src}, time.Hour, func(time.Time) string { called = true; return "x" })(ctx)
	if got != "" || called || src.stopped != 1 {
		t.Fatalf("got %q, fn called %v, stopped %d; want zero value, no call, ticker stopped", got, called, src.stopped)
	}
}

func TestAfterOnZeroDelayUsesTheSourceTime(t *testing.T) {
	src := &manualSource{now: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)}
	got := AfterOn(&Clock{Source: src}, 0, func(t time.Time) time.Time { return t })(context.Background())
	if !got.Equal(src.now) || len(src.periods) != 0 {
		t.Fatalf("got %v with %d tickers; want the Source's now and no ticker", got, len(src.periods))
	}
}

func TestAfterOnWithoutSourceIsAfter(t *testing.T) {
	for name, c := range map[string]*Clock{"nil clock": nil, "no source": NewClock(time.Second)} {
		start := time.Now()
		got := AfterOn(c, 5*time.Millisecond, func(time.Time) string { return "x" })(context.Background())
		if got != "x" || time.Since(start) < 5*time.Millisecond {
			t.Fatalf("%s: got %q after %v", name, got, time.Since(start))
		}
	}
}

func TestAfterWaitsThenProducesMsg(t *testing.T) {
	start := time.Now()
	got := After(5*time.Millisecond, func(time.Time) string { return "x" })(context.Background())
	if got != "x" || time.Since(start) < 5*time.Millisecond {
		t.Fatalf("got %q after %v", got, time.Since(start))
	}
}
