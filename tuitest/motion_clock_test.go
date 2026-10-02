package tuitest_test

import (
	"context"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/tuitest"
)

type motionModel struct {
	clk *motion.Clock
	n   *int
}

func (m motionModel) Init() tui.Cmd {
	c := m.clk.Start()
	if c == nil {
		return nil
	}
	return tui.FromCtx(func(ctx context.Context) tui.Msg { return c(ctx) })
}

func (m motionModel) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if _, ok := msg.(motion.TickMsg); ok {
		*m.n++
	}
	c := m.clk.Update(msg)
	if c == nil {
		return m, nil
	}
	return m, tui.FromCtx(func(ctx context.Context) tui.Msg { return c(ctx) })
}

func (m motionModel) View() string { return "motion" }

// Criterion #103: advancing a FakeClock by one interval delivers exactly one
// TickMsg to a running motion.Clock.
func TestAdvanceDrivesMotionClock(t *testing.T) {
	fc := tuitest.NewFakeClock()
	clk := motion.NewClock(100 * time.Millisecond)
	clk.Source = fc
	n := 0
	s := tuitest.New(motionModel{clk, &n}, 20, 3, tuitest.WithClock(fc))
	defer s.Close()
	s.Advance(100 * time.Millisecond)
	if n != 1 || clk.Frame() != 1 {
		t.Errorf("ticks = %d frame = %d, want 1", n, clk.Frame())
	}
	s.Advance(300 * time.Millisecond)
	if n != 4 {
		t.Errorf("ticks = %d, want 4", n)
	}
}

type afterMsg struct{ at time.Time }

type afterModel struct {
	clk *motion.Clock
	got *[]time.Time
}

func (m afterModel) Init() tui.Cmd {
	return tui.FromCtx(func(ctx context.Context) tui.Msg {
		return motion.AfterOn(m.clk, time.Hour, func(at time.Time) tui.Msg { return afterMsg{at} })(ctx)
	})
}

func (m afterModel) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if a, ok := msg.(afterMsg); ok {
		*m.got = append(*m.got, a.at)
	}
	return m, nil
}

func (m afterModel) View() string { return "after" }

// Criterion #102: a Cmd from motion.AfterOn on a Clock whose Source is the
// session's FakeClock fires when Advance passes its delay, not on the wall clock.
func TestAdvanceDrivesMotionAfterOn(t *testing.T) {
	fc := tuitest.NewFakeClock()
	clk := motion.NewClock(time.Second)
	clk.Source = fc
	var got []time.Time
	start := fc.Now()
	s := tuitest.New(afterModel{clk, &got}, 20, 3, tuitest.WithClock(fc))
	defer s.Close()
	s.Advance(59 * time.Minute)
	if len(got) != 0 {
		t.Fatalf("fired after 59m of a 1h delay: %v", got)
	}
	wall := time.Now()
	s.Advance(time.Minute)
	if len(got) != 1 || !got[0].Equal(start.Add(time.Hour)) {
		t.Fatalf("got %v, want one Msg at %v", got, start.Add(time.Hour))
	}
	if el := time.Since(wall); el > time.Second {
		t.Errorf("Advance took %v of wall time", el)
	}
}
