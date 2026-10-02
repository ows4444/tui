package clockview

import (
	"testing"
	"time"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	sw := New(ModeStopwatch)
	sw.elapsed = 65 * time.Second
	if got := sw.Linearize(); got != "Stopwatch, 00:01:05, stopped" {
		t.Errorf("stopwatch = %q", got)
	}
	sw.running = true
	if got := sw.Linearize(); got != "Stopwatch, 00:01:05, running" {
		t.Errorf("running stopwatch = %q", got)
	}
	tm := New(ModeTimer)
	tm.Duration = 90 * time.Second
	tm.elapsed = 30 * time.Second
	if got := tm.Linearize(); got != "Timer, 00:01:00 remaining, stopped" {
		t.Errorf("timer = %q", got)
	}
	ck := New(ModeClock)
	ck.Now = func() time.Time { return time.Date(2026, 1, 2, 15, 4, 5, 0, time.UTC) }
	ck.Layout = "15:04"
	if got := ck.Linearize(); got != "Clock, 15:04" {
		t.Errorf("clock = %q", got)
	}
}
