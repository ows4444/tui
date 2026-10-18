package main

import (
	"fmt"
	"time"
)

// watch is a stopwatch that keeps time against a clock it is given, so a test
// can drive it with a fake one. It never counts ticks: the time shown is always
// "now minus when it started", so a late or lost tick costs nothing.
type watch struct {
	running bool
	since   time.Time     // when the current run began
	before  time.Duration // time from earlier runs
	laps    []lap
}

// lap is one recorded lap: its length, and the total at the moment it was taken.
type lap struct {
	length, total time.Duration
}

func (w watch) elapsed(now time.Time) time.Duration {
	if w.running {
		return w.before + now.Sub(w.since)
	}
	return w.before
}

func (w watch) start(now time.Time) watch {
	if !w.running {
		w.running, w.since = true, now
	}
	return w
}

func (w watch) stop(now time.Time) watch {
	if w.running {
		w.before, w.running = w.elapsed(now), false
	}
	return w
}

// lap records the time since the last lap. It does nothing while stopped at zero.
func (w watch) lap(now time.Time) watch {
	total := w.elapsed(now)
	if total <= 0 {
		return w
	}
	var last time.Duration
	if n := len(w.laps); n > 0 {
		last = w.laps[n-1].total
	}
	w.laps = append(append([]lap(nil), w.laps...), lap{length: total - last, total: total})
	return w
}

func (w watch) reset() watch { return watch{} }

// extremes returns the indexes of the fastest and slowest lap, or -1 and -1 with
// fewer than two laps (one lap is neither).
func (w watch) extremes() (fast, slow int) {
	if len(w.laps) < 2 {
		return -1, -1
	}
	fast, slow = 0, 0
	for i, l := range w.laps {
		if l.length < w.laps[fast].length {
			fast = i
		}
		if l.length > w.laps[slow].length {
			slow = i
		}
	}
	return fast, slow
}

// stamp formats d as MM:SS.cc, or H:MM:SS.cc from an hour on.
func stamp(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	cs := int(d / (10 * time.Millisecond))
	h, m, s := cs/360000, cs/6000%60, cs/100%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs%100)
	}
	return fmt.Sprintf("%02d:%02d.%02d", m, s, cs%100)
}
