//go:build linux || darwin || dragonfly || freebsd || netbsd || openbsd

package tui

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/ows4444/tui/internal/termio"
)

// nextResize returns the first ResizeMsg on p.msgs, or fails after two seconds.
func nextResize(t *testing.T, p *Program) ResizeMsg {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case m := <-p.msgs:
			if r, ok := m.(ResizeMsg); ok {
				return r
			}
		case <-deadline:
			t.Fatal("no ResizeMsg: the resize was lost")
		}
	}
}

// A window-size signal that arrives after startResizeWatch returns, and
// before the goroutine that forwards resizes has run, still becomes a
// ResizeMsg. The listener used to be registered on that goroutine, and
// SIGWINCH is ignored when nothing listens, so a resize in a program's first
// moments was lost: TestResizeRepeatedlyEndsConsistent failed that way in CI.
func TestAResizeBeforeTheForwarderRunsIsNotLost(t *testing.T) {
	term := &termio.Fake{W: 80, H: 24, SizeKnown: true}
	p := NewProgram(staticModel{view: "v"}, WithTerminal(term))
	p.width, p.height = 80, 24
	watch := startResizeWatch(p)

	term.W, term.H = 100, 40
	if err := syscall.Kill(os.Getpid(), syscall.SIGWINCH); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond) // the forwarder is scheduled late

	done := make(chan struct{})
	finished := make(chan struct{})
	go func() { watch(done); close(finished) }()
	if got := nextResize(t, p); got != (ResizeMsg{Width: 100, Height: 40}) {
		t.Fatalf("ResizeMsg = %+v, want 100x40", got)
	}
	close(done)
	<-finished
}

// A resize from after the program read its size and before it began
// listening sends no signal the program can catch. startResizeWatch finds it
// by comparing the terminal's size with the one the program holds.
func TestAResizeBeforeListeningStartsIsNotLost(t *testing.T) {
	term := &termio.Fake{W: 100, H: 40, SizeKnown: true}
	p := NewProgram(staticModel{view: "v"}, WithTerminal(term))
	p.width, p.height = 80, 24 // what the program read a moment ago

	done := make(chan struct{})
	finished := make(chan struct{})
	watch := startResizeWatch(p)
	go func() { watch(done); close(finished) }()
	if got := nextResize(t, p); got != (ResizeMsg{Width: 100, Height: 40}) {
		t.Fatalf("ResizeMsg = %+v, want 100x40", got)
	}
	close(done)
	<-finished
}

// With the size unchanged and no signal, nothing is sent.
func TestStartResizeWatchSendsNothingForAnUnchangedSize(t *testing.T) {
	term := &termio.Fake{W: 80, H: 24, SizeKnown: true}
	p := NewProgram(staticModel{view: "v"}, WithTerminal(term))
	p.width, p.height = 80, 24
	done := make(chan struct{})
	finished := make(chan struct{})
	watch := startResizeWatch(p)
	go func() { watch(done); close(finished) }()
	select {
	case m := <-p.msgs:
		t.Fatalf("unexpected msg %#v for an unchanged size", m)
	case <-time.After(40 * time.Millisecond):
	}
	close(done)
	<-finished
}
