//go:build windows

package tui

import "time"

// resizePollFallback is how often the size is re-read in case the console
// never delivers a window-buffer-size event (for example when input comes
// from a WithInput source, which the console reader does not see). The
// primary source is the event, read from the console input queue.
const resizePollFallback = 250 * time.Millisecond

// watchResize sends a ResizeMsg when the console reports a window-buffer-size
// event (p.resizeKick, fed by the input reader) and, as a fallback, when a
// poll of GetConsoleScreenBufferInfo sees a different size.
func watchResize(p *Program, done <-chan struct{}) {
	watchResizeKicks(p, p.resizeKick, p.resizePollEvery(), done)
}
