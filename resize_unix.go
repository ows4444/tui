//go:build linux || darwin || dragonfly || freebsd || netbsd || openbsd

package tui

import (
	"os"
	"os/signal"
	"syscall"
)

// watchResize sends a ResizeMsg to p.msgs on every SIGWINCH until done is
// closed. When the queue is full the newest size is kept and sent as soon as
// there is room, so the model always ends up with the final size.
func watchResize(p *Program, done <-chan struct{}) {
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	defer signal.Stop(sigwinch)
	forwardResizes(p, sigwinch, p.termSize, done)
}

// forwardResizes delivers a coalesced ResizeMsg for each value on sig; see
// forwardResizeSignals.
func forwardResizes(p *Program, sig <-chan os.Signal, size func() (w, h int, ok bool), done <-chan struct{}) {
	forwardResizeSignals(p, sig, size, done)
}
