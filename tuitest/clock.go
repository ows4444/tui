package tuitest

import (
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/ows4444/tui"
)

// FakeClock is a tui.Clock that only moves when Session.Advance says so. Give
// it to a Session with WithClock; tui.Every then ticks on its time.
type FakeClock struct {
	mu      sync.Mutex
	now     time.Time
	tickers map[*fakeTicker]struct{}
	seq     int // tickers created so far; the next one's id
}

type fakeTicker struct {
	id      int
	d       time.Duration
	next    time.Time
	c       chan time.Time
	stopped chan struct{}
	once    sync.Once
}

// NewFakeClock returns a FakeClock reading a fixed start time.
func NewFakeClock() *FakeClock {
	return &FakeClock{
		now:     time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		tickers: map[*fakeTicker]struct{}{},
	}
}

// Now reports the fake time.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// NewTicker implements tui.Clock.
func (c *FakeClock) NewTicker(d time.Duration) (<-chan time.Time, func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTicker{id: c.seq, d: d, next: c.now.Add(d), c: make(chan time.Time), stopped: make(chan struct{})}
	c.seq++
	c.tickers[t] = struct{}{}
	return t.c, func() {
		t.once.Do(func() { close(t.stopped) })
		c.mu.Lock()
		delete(c.tickers, t)
		c.mu.Unlock()
	}
}

func (c *FakeClock) active() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.tickers)
}

// due pops the earliest tick at or before end, or reports none.
func (c *FakeClock) due(end time.Time) (*fakeTicker, time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ts := make([]*fakeTicker, 0, len(c.tickers))
	for t := range c.tickers {
		if !t.next.After(end) {
			ts = append(ts, t)
		}
	}
	if len(ts) == 0 {
		return nil, time.Time{}, false
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i].next.Before(ts[j].next) })
	t := ts[0]
	at := t.next
	t.next = at.Add(t.d)
	c.now = at
	return t, at, true
}

// WithClock runs the session's tui.Every on c. Move time with Advance.
func WithClock(c *FakeClock) tui.ProgramOption { return tui.WithClock(c) }

// Advance moves the session's FakeClock forward by d, delivering each tick
// that falls due in order and returning once the model has handled them all (the screen may still be
// drawing; Keys, Send and Screen settle it).
// It waits on the model, not on the wall clock. It does nothing when the
// session has no FakeClock.
func (s *Session) Advance(d time.Duration) {
	c, _ := s.prog.Clock().(*FakeClock)
	if c == nil {
		return
	}
	// A ticker is registered when the Program's loop handles the Every Cmd
	// from Init; give it a moment rather than race it.
	for i := 0; i < 1000 && c.active() == 0; i++ {
		time.Sleep(100 * time.Microsecond)
	}
	end := c.Now().Add(d)
	var fired int64
	start := s.updates.Load()
	for {
		t, at, ok := c.due(end)
		if !ok {
			break
		}
		select {
		case t.c <- at:
			fired++
		case <-t.stopped:
		case <-s.done:
			return
		}
	}
	c.mu.Lock()
	c.now = end
	c.mu.Unlock()
	// Wait for the model's Updates, not for the output to go quiet: that is
	// what keeps Advance off the wall clock.
	for s.updates.Load() < start+fired {
		select {
		case <-s.done:
			return
		default:
			runtime.Gosched()
		}
	}
}

// fire delivers one tick of the ticker created id'th (counting from 0),
// setting the fake time to when it was due. It waits briefly for the ticker to
// be created, and reports false if it never was, was stopped, or done closed.
func (c *FakeClock) fire(id int, done <-chan struct{}) bool {
	var t *fakeTicker
	var at time.Time
	for i := 0; i < 5000 && t == nil; i++ {
		c.mu.Lock()
		for k := range c.tickers {
			if k.id == id {
				t, at = k, k.next
				k.next = at.Add(k.d)
				if at.After(c.now) {
					c.now = at
				}
			}
		}
		c.mu.Unlock()
		if t == nil {
			time.Sleep(100 * time.Microsecond)
		}
	}
	if t == nil {
		return false
	}
	select {
	case t.c <- at:
		return true
	case <-t.stopped:
	case <-done:
	}
	return false
}
