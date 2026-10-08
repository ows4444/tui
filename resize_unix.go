//go:build linux || darwin || dragonfly || freebsd || netbsd || openbsd

package tui

import (
	"os"
	"os/signal"
	"syscall"
)

// startResizeWatch begins listening for SIGWINCH before it returns, and
// returns the function that sends a ResizeMsg to p.msgs for each one until
// done is closed. When the queue is full the newest size is kept and sent as
// soon as there is room, so the model always ends up with the final size.
//
// Listening starts here, on the caller's goroutine, not in the returned
// function: that one runs on a goroutine of its own, and a resize in the
// moment before it was scheduled used to find no listener. SIGWINCH is ignored
// by default, so that resize was lost and the program stayed drawn at its old
// size until the next one. A resize from before this call, after the program
// first read its size, is caught by comparing the two sizes.
func startResizeWatch(p *Program) func(done <-chan struct{}) {
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	if w, h, ok := p.termSize(); ok && (w != p.width || h != p.height) {
		select {
		case sigwinch <- syscall.SIGWINCH:
		default: // one is already waiting; it will read the same size
		}
	}
	return func(done <-chan struct{}) {
		defer signal.Stop(sigwinch)
		forwardResizes(p, sigwinch, p.termSize, done)
	}
}

// forwardResizes delivers a coalesced ResizeMsg for each value on sig; see
// forwardResizeSignals.
func forwardResizes(p *Program, sig <-chan os.Signal, size func() (w, h int, ok bool), done <-chan struct{}) {
	forwardResizeSignals(p, sig, size, done)
}
