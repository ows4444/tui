package tui

import "time"

// forwardResizeSignals turns each value on sig into the current terminal size
// (from size) and delivers it to p.msgs. At most one size waits for queue room,
// and a newer one replaces it: a burst of signals against a full queue
// coalesces into a single ResizeMsg holding the last size, never a dropped one.
// It is the one coalescing path every platform's resize source feeds (SIGWINCH
// on unix, console events and a fallback poll on windows).
func forwardResizeSignals[T any](p *Program, sig <-chan T, size func() (w, h int, ok bool), done <-chan struct{}) {
	var pending Msg
	for {
		var out chan<- Msg
		if pending != nil {
			out = p.msgs
		}
		select {
		case _, open := <-sig:
			if !open {
				sig = nil // a closed source is a source that has no more to say
				continue
			}
			if w, h, ok := size(); ok {
				pending = ResizeMsg{Width: w, Height: h}
			}
		case out <- pending:
			pending = nil
		case <-done:
			return
		}
	}
}

// changedSize wraps size so it reports ok only when the size differs from the
// last one it reported. The console queues a resize event for changes that do
// not alter the window (a buffer resize, a scroll), and the fallback poll
// fires on a timer, so neither may turn into a ResizeMsg by itself.
func changedSize(size func() (w, h int, ok bool), lastW, lastH int) func() (int, int, bool) {
	return func() (int, int, bool) {
		w, h, ok := size()
		if !ok || (w == lastW && h == lastH) {
			return 0, 0, false
		}
		lastW, lastH = w, h
		return w, h, true
	}
}

// watchResizeKicks delivers a ResizeMsg whenever the terminal size changed
// after a value arrives on kick (a console window-buffer-size event, the
// primary source) or after every pollEvery (a fallback for consoles that do not
// report the event; 0 disables it), until done is closed.
func watchResizeKicks(p *Program, kick <-chan struct{}, pollEvery time.Duration, done <-chan struct{}) {
	w, h, _ := p.termSize()
	size := changedSize(p.termSize, w, h)

	merged := make(chan struct{}, 1)
	var tick <-chan time.Time
	if pollEvery > 0 {
		t := time.NewTicker(pollEvery)
		defer t.Stop()
		tick = t.C
	}
	stopped := make(chan struct{})
	defer func() { <-stopped }()
	go func() {
		defer close(stopped)
		for {
			select {
			case <-kick:
			case <-tick:
			case <-done:
				return
			}
			select {
			case merged <- struct{}{}:
			default: // one is already pending; it will read the latest size
			}
		}
	}()
	forwardResizeSignals(p, merged, size, done)
}

// resizePollEvery returns the configured resize polling interval, or the
// 250ms default when WithResizePoll was not given or d <= 0.
func (p *Program) resizePollEvery() time.Duration {
	if p.resizePoll > 0 {
		return p.resizePoll
	}
	return 250 * time.Millisecond
}
