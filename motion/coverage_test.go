package motion

import (
	"context"
	"testing"
	"time"
)

func TestSpringTargetReports(t *testing.T) {
	s := NewSpring(0, 100)
	s.SetTarget(5)
	if s.Target() != 5 {
		t.Errorf("Target = %v, want 5", s.Target())
	}
}

func TestNilClockReadsTheWallClockAndFrameZero(t *testing.T) {
	var c *Clock
	before := time.Now()
	if got := c.Now(); got.Before(before) {
		t.Errorf("Now() = %v, before %v", got, before)
	}
	if c.Frame() != 0 {
		t.Errorf("nil Clock Frame = %d", c.Frame())
	}
	if NewClock(time.Millisecond).Now().IsZero() {
		t.Error("Clock without a Source returned the zero time")
	}
}

// fakeSource is a Source whose ticker the test fires by hand.
type fakeSource struct {
	now     time.Time
	ch      chan time.Time
	stopped bool
}

func (f *fakeSource) Now() time.Time { return f.now }

func (f *fakeSource) NewTicker(time.Duration) (<-chan time.Time, func()) {
	f.ch = make(chan time.Time, 1)
	return f.ch, func() { f.stopped = true }
}

func TestClockWithASourceTicksFromItsTickerAndStopsIt(t *testing.T) {
	src := &fakeSource{now: time.Unix(100, 0)}
	c := NewClock(time.Second)
	c.Source = src

	first := c.Start()
	if first == nil {
		t.Fatal("Start returned no Cmd")
	}
	if c.Now() != src.now {
		t.Errorf("Now = %v, want the source's %v", c.Now(), src.now)
	}
	at := time.Unix(101, 0)
	src.ch <- at
	msg, ok := first(context.Background()).(TickMsg)
	if !ok || !msg.At.Equal(at) {
		t.Fatalf("first tick = %#v", msg)
	}
	next := c.Update(msg)
	if next == nil || c.Frame() != 1 {
		t.Fatalf("Update: next %v frame %d", next != nil, c.Frame())
	}
	c.Stop()
	c.Start() // a new run replaces the ticker and stops the old one
	if !src.stopped {
		t.Error("the Source ticker of the ended run was not stopped")
	}
}

func TestSourceTickWaitEndsOnContextCancel(t *testing.T) {
	src := &fakeSource{now: time.Unix(0, 0)}
	c := NewClock(time.Second)
	c.Source = src
	cmd := c.Start()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := cmd(ctx); got != nil {
		t.Errorf("a cancelled wait returned %v", got)
	}
}

func TestTimelineDuration(t *testing.T) {
	tl := &Timeline{}
	tl.At(100*time.Millisecond, Tween{From: 0, To: 1, Duration: 400 * time.Millisecond}, func(float64) {})
	if got := tl.Duration(); got != 500*time.Millisecond {
		t.Errorf("Duration = %v, want 500ms", got)
	}
}
