package tui

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/ows4444/tui/ansi"
)

// capturePendingMax is how many captured lines wait for the event loop before
// the oldest are dropped, so a program that prints in a tight loop cannot
// exhaust memory.
const capturePendingMax = 10000

// captureChunk is the longest piece of a line read at once; a longer line is
// cut into pieces of this size.
const captureChunk = 64 << 10

type capturedLine struct {
	text   string
	stderr bool
}

// capture redirects os.Stdout, os.Stderr and the standard logger into the
// Program while it runs; see WithCaptureOutput.
type capture struct {
	p                *Program
	origOut, origErr *os.File
	wOut, wErr       *os.File
	restoreLog       func()

	mu      sync.Mutex
	pending []capturedLine
	dropped int
	wake    chan struct{}
	readers sync.WaitGroup
	pumped  chan struct{}
	stop    chan struct{}
}

// startCapture installs the capture and returns the function that removes it.
// It returns a no-op if the pipes cannot be made: the program then runs without
// capture, as if the option were off.
func (p *Program) startCapture() func() {
	rOut, wOut, err := os.Pipe()
	if err != nil {
		return func() {}
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		_ = rOut.Close()
		_ = wOut.Close()
		return func() {}
	}
	c := &capture{
		p: p, origOut: os.Stdout, origErr: os.Stderr, wOut: wOut, wErr: wErr,
		wake: make(chan struct{}, 1), pumped: make(chan struct{}), stop: make(chan struct{}),
	}
	os.Stdout, os.Stderr = wOut, wErr
	p.cap.Store(c)
	// The standard logger holds the os.Stderr it was created with, not the
	// variable, so it needs redirecting too, but only if nothing else has.
	if log.Writer() == io.Writer(c.origErr) {
		log.SetOutput(wErr)
		c.restoreLog = func() { log.SetOutput(c.origErr) }
	}
	c.readers.Add(2)
	go c.read(rOut, false)
	go c.read(rErr, true)
	go c.pump()
	return c.close
}

// read turns what is written to r into lines.
func (c *capture) read(r *os.File, stderr bool) {
	defer c.readers.Done()
	defer r.Close()
	br := bufio.NewReaderSize(r, 4096)
	var line []byte
	for {
		chunk, prefix, err := br.ReadLine()
		line = append(line, chunk...)
		if (!prefix || len(line) >= captureChunk) && (len(line) > 0 || err == nil) {
			c.add(string(line), stderr)
			line = line[:0]
		}
		if err != nil {
			if len(line) > 0 {
				c.add(string(line), stderr)
			}
			return
		}
	}
}

func (c *capture) add(text string, stderr bool) {
	c.mu.Lock()
	if len(c.pending) >= capturePendingMax {
		c.pending = c.pending[1:]
		c.dropped++
	}
	c.pending = append(c.pending, capturedLine{text: ansi.SanitizeKeepSGR(text), stderr: stderr})
	c.mu.Unlock()
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

// take removes and returns the waiting lines, with a notice first if some were
// dropped.
func (c *capture) take() []capturedLine {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.pending
	c.pending = nil
	if c.dropped > 0 {
		out = append([]capturedLine{{text: fmt.Sprintf("tui: %d captured lines were dropped", c.dropped), stderr: true}}, out...)
		c.dropped = 0
	}
	return out
}

// pump commits the captured lines through the Program, never blocking: a line
// the event loop has no room for waits, so a program that prints from Update
// cannot deadlock against a full queue.
func (c *capture) pump() {
	defer close(c.pumped)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	var backlog []capturedLine
	for {
		backlog = append(backlog, c.take()...)
		for len(backlog) > 0 && c.p.TrySend(printMsg{text: backlog[0].text, stderr: backlog[0].stderr}) {
			backlog = backlog[1:]
		}
		select {
		case <-c.stop:
			// The loop is over, so the rest goes to the real files, after the
			// terminal has been put back.
			for _, l := range append(backlog, c.take()...) {
				f := c.origOut
				if l.stderr {
					f = c.origErr
				}
				_, _ = fmt.Fprintln(f, l.text)
			}
			return
		case <-c.wake:
		case <-tick.C:
		}
	}
}

// pause gives os.Stdout and os.Stderr back to the terminal, for a Suspend
// function that hands it to another program; resume catches them again.
func (c *capture) pause() { os.Stdout, os.Stderr = c.origOut, c.origErr }

func (c *capture) resume() { os.Stdout, os.Stderr = c.wOut, c.wErr }

// close puts the real files back and flushes what is left.
func (c *capture) close() {
	c.p.cap.Store(nil)
	os.Stdout, os.Stderr = c.origOut, c.origErr
	if c.restoreLog != nil {
		c.restoreLog()
	}
	_ = c.wOut.Close()
	_ = c.wErr.Close()
	c.readers.Wait()
	close(c.stop)
	<-c.pumped
}
