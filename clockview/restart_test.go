package clockview

import (
	"context"
	"testing"
	"time"

	"github.com/ows4444/tui"
)

// minTicks is the fewest ticks a single chain must deliver in 200 ms at 20 ms.
// Ten are due; five leaves room for a loaded machine yet is far from zero.
const minTicks = 5

// drive runs m the way a Program would: each Cmd runs on its own goroutine
// and its Msg comes back to Update, for total wall time. It returns the model.
func drive(m Model, start func(*Model) []tui.Cmd, total time.Duration) Model {
	msgs := make(chan tui.Msg, 256)
	run := func(c tui.Cmd) {
		if c != nil {
			go func() { msgs <- tui.RunCmd(context.Background(), c) }()
		}
	}
	for _, c := range start(&m) {
		run(c)
	}
	deadline := time.After(total)
	for {
		select {
		case msg := <-msgs:
			var c tui.Cmd
			m, c = m.Update(msg)
			run(c)
		case <-deadline:
			return m
		}
	}
}

// #32: a 20 ms Clock stopped and restarted within one interval, run for
// 200 ms, receives at most 11 ticks, not the ~20 that two live chains give. A
// loaded machine (-race, parallel packages) can only delay ticks, never add
// them, so the lower bound is loose and the upper bound is what tells one chain
// from two.
func TestStopThenStartWithinOneIntervalKeepsASingleTickChain(t *testing.T) {
	m := New(ModeStopwatch)
	m.Interval = 20 * time.Millisecond
	m = drive(m, func(m *Model) []tui.Cmd {
		first := m.Start()
		m.Stop()
		return []tui.Cmd{first, m.Start()}
	}, 200*time.Millisecond)
	ticks := int(m.elapsed / m.Interval)
	if ticks < minTicks || ticks > 11 {
		t.Fatalf("received %d ticks in 200ms at 20ms, want %d-11", ticks, minTicks)
	}
}

func TestStartOnARunningModelDoesNotDoubleTheRate(t *testing.T) {
	m := New(ModeStopwatch)
	m.Interval = 20 * time.Millisecond
	m = drive(m, func(m *Model) []tui.Cmd { return []tui.Cmd{m.Start(), m.Start()} }, 200*time.Millisecond)
	if ticks := int(m.elapsed / m.Interval); ticks < minTicks || ticks > 11 {
		t.Fatalf("received %d ticks, want %d-11", ticks, minTicks)
	}
}

func TestStaleTickAfterStopIsDroppedEvenWhenRunningAgain(t *testing.T) {
	m := New(ModeStopwatch)
	m.Interval = time.Millisecond
	first := m.Start()
	stale := tui.RunCmd(context.Background(), first).(tickMsg)
	m.Stop()
	m.Start()
	if next, cmd := m.Update(stale); cmd != nil || next.elapsed != 0 {
		t.Fatalf("a tick from the stopped chain was applied: elapsed=%v cmd=%v", next.elapsed, cmd != nil)
	}
}
