package tui

import (
	"testing"
	"time"
)

func TestChangedSize(t *testing.T) {
	cur := [2]int{80, 24}
	ok := true
	f := changedSize(func() (int, int, bool) { return cur[0], cur[1], ok }, 80, 24)
	if _, _, got := f(); got {
		t.Fatal("unchanged size reported")
	}
	cur = [2]int{100, 30}
	if w, h, got := f(); !got || w != 100 || h != 30 {
		t.Fatalf("changed size = %d,%d,%v", w, h, got)
	}
	if _, _, got := f(); got {
		t.Fatal("same size reported twice")
	}
	ok = false
	cur = [2]int{1, 1}
	if _, _, got := f(); got {
		t.Fatal("failed size read reported")
	}
}

// A console resize event (a kick) reaches the model as a ResizeMsg well inside
// 50 ms, with the poll fallback disabled so only the event path can deliver it.
func TestKickDeliversResizeMsgQuickly(t *testing.T) {
	p := NewProgram(staticModel{view: "v"})
	cur := [2]int{80, 24}
	kick := make(chan struct{}, 1)
	done := make(chan struct{})
	defer close(done)
	size := changedSize(func() (int, int, bool) { return cur[0], cur[1], true }, 80, 24)
	go forwardResizeSignals(p, kick, size, done)

	cur = [2]int{120, 40}
	start := time.Now()
	kick <- struct{}{}
	select {
	case m := <-p.msgs:
		if m != (ResizeMsg{Width: 120, Height: 40}) {
			t.Fatalf("msg = %#v", m)
		}
		if d := time.Since(start); d > 50*time.Millisecond {
			t.Fatalf("ResizeMsg took %v, want < 50ms", d)
		}
	case <-time.After(time.Second):
		t.Fatal("no ResizeMsg after a kick")
	}

	// A kick that does not change the size produces nothing.
	kick <- struct{}{}
	select {
	case m := <-p.msgs:
		t.Fatalf("unexpected msg %#v for an unchanged size", m)
	case <-time.After(30 * time.Millisecond):
	}
}

// watchResizeKicks stops when done closes, with and without the poll.
func TestWatchResizeKicksStops(t *testing.T) {
	for _, poll := range []time.Duration{0, time.Millisecond} {
		p := NewProgram(staticModel{view: "v"})
		done := make(chan struct{})
		finished := make(chan struct{})
		go func() { watchResizeKicks(p, make(chan struct{}), poll, done); close(finished) }()
		time.Sleep(5 * time.Millisecond)
		close(done)
		select {
		case <-finished:
		case <-time.After(time.Second):
			t.Fatalf("watchResizeKicks(poll=%v) did not stop", poll)
		}
	}
}

func TestKickResizeNeverBlocks(t *testing.T) {
	p := NewProgram(staticModel{view: "v"})
	for i := 0; i < 10; i++ {
		p.kickResize()
	}
	if len(p.resizeKick) != 1 {
		t.Fatalf("pending kicks = %d, want 1", len(p.resizeKick))
	}
	var zero Program
	zero.kickResize() // nil channel: must not block
}
