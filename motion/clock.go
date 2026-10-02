package motion

import (
	"context"
	"time"
)

// TickMsg is the Msg a Clock sends once per interval.
type TickMsg struct {
	Clock *Clock
	// At is when the tick was scheduled to fire.
	At  time.Time
	gen uint64
}

// Source is an injectable time source. tuitest.FakeClock and any tui.Clock
// that also reports Now implement it; the wall clock is used when a Clock
// has none.
type Source interface {
	Now() time.Time
	// NewTicker returns a channel that receives the time every d and a
	// func that stops it.
	NewTicker(d time.Duration) (<-chan time.Time, func())
}

// Clock is one shared animation tick. Widgets that opt in (spinner,
// skeleton and loadingbar have a Clock field) render from Frame instead of
// scheduling their own tick chain, so every widget on one Clock shows the
// same frame at every tick and a program issues one tick Cmd per interval
// however many widgets animate.
//
// A Clock is used by pointer and is not safe for concurrent use; drive it
// from the Update loop: return Start's Cmd once, and pass every Msg to
// Update, returning the Cmd it produces.
type Clock struct {
	Interval time.Duration
	// Motion is the reduced-motion preference. With Reduced, Start and
	// Update schedule nothing and Frame stays 0.
	Motion Preference
	// Source, when set, supplies time and ticks instead of the wall clock,
	// so a test drives the Clock by advancing a fake. Set it before Start.
	Source Source

	ticks <-chan time.Time // Source ticker for the current run
	stopT func()
	quit  chan struct{} // closed when the current run ends

	frame   int
	running bool
	// gen invalidates in-flight ticks on Start and Stop so a restart
	// within one interval cannot double the chain.
	gen   uint64
	start time.Time
	n     int // ticks scheduled since start; tick n fires at start + n*Interval
	users int // animators holding the Clock through Acquire
}

// NewClock returns a stopped Clock ticking every interval.
func NewClock(interval time.Duration) *Clock { return &Clock{Interval: interval} }

// Now reports the Clock's time: Source.Now when a Source is set, else the
// wall clock. A nil Clock reads the wall clock.
func (c *Clock) Now() time.Time {
	if c == nil || c.Source == nil {
		return time.Now()
	}
	return c.Source.Now()
}

// begin starts a run: the Source ticker, when there is a Source.
func (c *Clock) begin() {
	c.end()
	c.running = true
	c.gen++
	c.start, c.n = c.Now(), 0
	if c.Source != nil && c.Interval > 0 {
		c.ticks, c.stopT = c.Source.NewTicker(c.Interval)
		c.quit = make(chan struct{})
	}
}

// end releases the Source ticker of the run that is ending.
func (c *Clock) end() {
	if c.stopT != nil {
		c.stopT()
		close(c.quit)
	}
	c.ticks, c.stopT, c.quit = nil, nil, nil
}

// Frame reports how many ticks the Clock has delivered. A nil Clock is
// at frame 0.
func (c *Clock) Frame() int {
	if c == nil {
		return 0
	}
	return c.frame
}

// tick waits until the next slot at start + n*Interval, then yields a
// TickMsg. Scheduling against the start instead of the last tick keeps the
// rate independent of how long Update takes; a slot already missed is
// skipped rather than fired in a burst. The wait ends when ctx is cancelled
// (Run returned).
func (c *Clock) tick() func(context.Context) any {
	if c.ticks != nil {
		ticks, quit, gen := c.ticks, c.quit, c.gen
		return func(ctx context.Context) any {
			select {
			case at := <-ticks:
				return TickMsg{Clock: c, At: at, gen: gen}
			case <-quit:
			case <-ctx.Done():
			}
			return nil
		}
	}
	c.n++
	deadline := c.start.Add(time.Duration(c.n) * c.Interval)
	if now := time.Now(); c.Interval > 0 && deadline.Before(now) {
		c.n = int(now.Sub(c.start)/c.Interval) + 1
		deadline = c.start.Add(time.Duration(c.n) * c.Interval)
	}
	gen := c.gen
	return func(ctx context.Context) any {
		if !waitCtx(ctx, time.Until(deadline)) {
			return nil
		}
		return TickMsg{Clock: c, At: deadline, gen: gen}
	}
}

// Start begins ticking and returns the first tick as a context-carrying Cmd:
// wrap it with tui.FromCtx. The pending wait ends when ctx is cancelled (Run
// returned). It returns nil under reduced motion or on a nil Clock, and
// starting a running Clock returns nil so a second Start cannot double the
// tick chain.
func (c *Clock) Start() func(context.Context) any {
	if c == nil || c.Motion.Reduced() || c.running {
		return nil
	}
	c.begin()
	return c.tick()
}

// Stop ends ticking; the pending tick, if any, is absorbed by Update.
func (c *Clock) Stop() {
	if c != nil {
		c.running = false
		c.gen++
		c.end()
	}
}

// Running reports whether the Clock is ticking.
func (c *Clock) Running() bool { return c != nil && c.running }

// Update advances the frame on this Clock's TickMsg and returns the next tick
// as a context-carrying Cmd (see Start). Any other Msg, or a tick after Stop,
// is a no-op.
func (c *Clock) Update(msg any) func(context.Context) any {
	t, ok := msg.(TickMsg)
	if c == nil || !ok || t.Clock != c || !c.running || t.gen != c.gen || c.Motion.Reduced() {
		return nil
	}
	c.frame++
	return c.tick()
}

// DefaultFPS is the frame rate the program renders non-input Msgs at, and
// so the fastest rate a frame-synced Clock ticks.
const DefaultFPS = 60

// FrameInterval is the interval between frames at fps; fps <= 0 means
// DefaultFPS. Pass the same fps given to tui.WithMaxFPS so animation ticks
// line up with the renderer's frame budget.
func FrameInterval(fps int) time.Duration {
	if fps <= 0 {
		fps = DefaultFPS
	}
	return time.Second / time.Duration(fps)
}

// NewFrameClock returns a stopped Clock ticking once per frame at fps.
// Animating widgets Acquire and Release it, so it ticks only while
// something animates.
func NewFrameClock(fps int) *Clock { return NewClock(FrameInterval(fps)) }

// Acquire registers one animator and returns the Cmd that starts the
// Clock if this is the first, else nil: however many widgets animate there
// is one tick chain, so at most one tick per frame interval. Pair every
// Acquire with a Release.
func (c *Clock) Acquire() func(context.Context) any {
	if c == nil {
		return nil
	}
	c.users++
	return c.Start()
}

// Release drops one animator and stops the Clock when none remain, so an
// idle program schedules no timer. Extra Releases are ignored.
func (c *Clock) Release() {
	if c == nil || c.users == 0 {
		return
	}
	if c.users--; c.users == 0 {
		c.Stop()
	}
}

// waitCtx blocks for d and reports whether it ran to the end; cancelling ctx
// (Run returned) ends the wait early.
func waitCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// After returns a context-carrying Cmd that waits d and then produces fn's Msg,
// the one-shot timer behind blink, dismiss and timeout Cmds: wrap it with
// tui.FromCtx. The wait ends when ctx is cancelled (Run returned), and fn is
// then not called and no Msg is sent. T is the program's Msg type, so
// tui.FromCtx(After(d, fn)) is a tui.Cmd when fn returns tui.Msg.
func After[T any](d time.Duration, fn func(time.Time) T) func(context.Context) T {
	return func(ctx context.Context) T {
		var zero T
		if !waitCtx(ctx, d) {
			return zero
		}
		return fn(time.Now())
	}
}

// AfterOn is After on c's time. When c has a Source (a test's
// tuitest.FakeClock), the wait is the first tick of a Source ticker of period
// d, created when the Cmd runs and stopped when it returns, and fn gets that
// tick's time; advancing the fake past d ends the wait without touching the
// wall clock. A nil c, or one without a Source, makes it After. The Source is
// read when AfterOn is called, as a Clock is driven from the Update loop.
func AfterOn[T any](c *Clock, d time.Duration, fn func(time.Time) T) func(context.Context) T {
	if c == nil || c.Source == nil {
		return After(d, fn)
	}
	src := c.Source
	return func(ctx context.Context) T {
		var zero T
		if d <= 0 {
			return fn(src.Now())
		}
		ticks, stop := src.NewTicker(d)
		defer stop()
		select {
		case at := <-ticks:
			return fn(at)
		case <-ctx.Done():
			return zero
		}
	}
}
