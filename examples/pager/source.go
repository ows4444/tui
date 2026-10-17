package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

const (
	maxLineBytes = 1 << 20 // a longer line is cut
	batchLines   = 512     // most lines handed to the screen at once
	pollEvery    = 200 * time.Millisecond
)

// item is what the reading goroutine hands over: a line, or the end.
type item struct {
	line string
	err  error
}

// batchMsg carries lines that arrived together, and the channel to read on.
type batchMsg struct {
	lines []string
	ch    <-chan item
}

// eofMsg says the source has ended, with the error that ended it, if any.
type eofMsg struct{ err error }

// readLines sends the lines of r to ch and closes it at the end. At the end of
// r it stops, unless follow reports true, in which case it waits for more (the
// way tail -f does) until ctx is cancelled. A line longer than maxLineBytes is
// cut.
func readLines(ctx context.Context, r io.Reader, follow func() bool, ch chan<- item) {
	defer close(ch)
	br := bufio.NewReaderSize(r, 64<<10)
	var cur []byte
	send := func(l string) bool {
		select {
		case ch <- item{line: ansi.Sanitize(strings.ReplaceAll(l, "\t", "    "))}:
			return true
		case <-ctx.Done():
			return false
		}
	}
	for {
		chunk, err := br.ReadSlice('\n')
		if len(cur) < maxLineBytes {
			cur = append(cur, chunk...)
		}
		switch {
		case err == bufio.ErrBufferFull:
			continue // a long line arrives in pieces
		case err == io.EOF:
			if follow() { // a file being written: the line may not be finished
				select {
				case <-time.After(pollEvery):
					continue
				case <-ctx.Done():
					return
				}
			}
			if len(cur) > 0 {
				send(strings.TrimRight(string(cur), "\r\n"))
			}
			return
		case err != nil:
			ch <- item{err: err}
			return
		}
		if !send(strings.TrimRight(string(cur), "\r\n")) {
			return
		}
		cur = cur[:0]
	}
}

// next returns a Cmd that waits for lines: the first to arrive, and any that
// are already waiting behind it, up to batchLines.
func next(ch <-chan item) tui.Cmd {
	return func() tui.Msg {
		first, ok := <-ch
		if !ok {
			return eofMsg{}
		}
		if first.err != nil {
			return eofMsg{err: first.err}
		}
		lines := []string{first.line}
		for len(lines) < batchLines {
			select {
			case it, ok := <-ch:
				if !ok {
					return batchMsg{lines: lines, ch: ch}
				}
				if it.err != nil {
					return batchMsg{lines: lines, ch: ch}
				}
				lines = append(lines, it.line)
			default:
				return batchMsg{lines: lines, ch: ch}
			}
		}
		return batchMsg{lines: lines, ch: ch}
	}
}

// startReading begins reading r and returns the Cmd for its first lines, and
// the function that stops it.
func startReading(r io.Reader, follow *atomic.Bool) (tui.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan item, 2*batchLines)
	go readLines(ctx, r, follow.Load, ch)
	return next(ch), cancel
}

// sample is the text the pager opens when it is given no file and nothing is
// piped in.
func sample() string {
	var b strings.Builder
	b.WriteString("PAGER SAMPLE\n============\n\nThis text stands in for a file. Press / to search, w to wrap, # for line\nnumbers, F to follow, ? for help.\n\n")
	for i := 1; i <= 12; i++ {
		fmt.Fprintf(&b, "Section %d\n---------\n", i)
		for j := 1; j <= 8; j++ {
			fmt.Fprintf(&b, "  %2d.%d  The quick brown fox number %d jumps over the lazy dog, then reads a long line that is wide enough to need wrapping or sideways scrolling in a narrow window.\n", i, j, i*10+j)
		}
		b.WriteString("\n")
	}
	b.WriteString("END OF SAMPLE\n")
	return b.String()
}
