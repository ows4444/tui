//go:build linux || darwin || dragonfly || freebsd || netbsd || openbsd

package tui

import (
	"os"
	"syscall"
	"testing"
	"time"
)

// #21: 200 resizes arrive while the queue is full; once there is room, the
// model gets a ResizeMsg with the final size.
func TestResizeBurstAgainstFullQueueDeliversFinalSize(t *testing.T) {
	p := NewProgram(staticModel{view: "v"})
	for len(p.msgs) < cap(p.msgs) {
		p.msgs <- struct{}{} // the loop is busy: the queue is full
	}
	sig := make(chan os.Signal)
	done := make(chan struct{})
	finished := make(chan struct{})
	n := 0
	size := func() (int, int, bool) { n++; return 100 + n, 50 + n, true }
	go func() { forwardResizes(p, sig, size, done); close(finished) }()

	for i := 0; i < 200; i++ {
		select {
		case sig <- syscall.SIGWINCH:
		case <-time.After(2 * time.Second):
			t.Fatalf("signal %d not accepted: the watcher blocked on the full queue", i)
		}
	}

	var last ResizeMsg
	got := false
	deadline := time.After(2 * time.Second)
	for !got || last.Width != 300 {
		select {
		case m := <-p.msgs: // the loop catching up
			if r, ok := m.(ResizeMsg); ok {
				last, got = r, true
			}
		case <-deadline:
			t.Fatalf("last ResizeMsg = %+v (seen %v), want 300x250", last, got)
		}
	}
	if last != (ResizeMsg{Width: 300, Height: 250}) {
		t.Fatalf("last ResizeMsg = %+v, want 300x250", last)
	}
	close(done)
	<-finished
}

// Nothing is sent when the terminal size cannot be read.
func TestResizeWithoutSizeSendsNothing(t *testing.T) {
	p := NewProgram(staticModel{view: "v"})
	sig := make(chan os.Signal)
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		forwardResizes(p, sig, func() (int, int, bool) { return 0, 0, false }, done)
		close(finished)
	}()
	sig <- syscall.SIGWINCH
	close(done)
	<-finished
	if len(p.msgs) != 0 {
		t.Fatalf("%d messages queued, want none", len(p.msgs))
	}
}
