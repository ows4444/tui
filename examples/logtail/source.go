package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

const (
	maxLineBytes = 1 << 20
	batchLines   = 512
	pollEvery    = 200 * time.Millisecond
)

type item struct {
	line string
	err  error
}

// linesMsg carries lines that arrived together, and the channel for the next.
type linesMsg struct {
	lines []string
	ch    <-chan item
}

type eofMsg struct{ err error }

// readLines sends the lines of r to ch and closes it at the end. At the end of
// r it waits for more, as tail -f does, if follow is true, until ctx ends.
func readLines(ctx context.Context, r io.Reader, follow bool, ch chan<- item) {
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
			continue
		case err == io.EOF:
			if follow { // the line may not be finished yet
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

// next returns a Cmd that waits for lines: the first to arrive and any already
// waiting behind it.
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
				if !ok || it.err != nil {
					return linesMsg{lines: lines, ch: ch}
				}
				lines = append(lines, it.line)
			default:
				return linesMsg{lines: lines, ch: ch}
			}
		}
		return linesMsg{lines: lines, ch: ch}
	}
}

// start begins reading r and returns the Cmd for its first lines and the
// function that stops it.
func start(r io.Reader, follow bool) (tui.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan item, 2*batchLines)
	go readLines(ctx, r, follow, ch)
	return next(ch), cancel
}

// ---- the made-up service ---------------------------------------------------

// generator makes the log of an imaginary web service, forever: mostly JSON,
// some plain text, a few warnings, now and then a burst of errors.
type generator struct {
	rng  *rand.Rand
	n    int
	base time.Time
}

func newGenerator(base time.Time) *generator {
	return &generator{rng: rand.New(rand.NewSource(11)), base: base} // #nosec G404 -- made-up log lines
}

var (
	paths   = []string{"/api/orders", "/api/users", "/api/search", "/healthz", "/static/app.js", "/api/cart"}
	methods = []string{"GET", "GET", "GET", "POST", "PUT", "DELETE"}
)

// lines makes the next k lines.
func (g *generator) lines(k int) []string {
	var out []string
	for i := 0; i < k; i++ {
		g.n++
		ts := g.base.Add(time.Duration(g.n) * 137 * time.Millisecond).Format("2006-01-02T15:04:05.000Z")
		r := g.rng.Intn(100)
		path, method := paths[g.rng.Intn(len(paths))], methods[g.rng.Intn(len(methods))]
		ms := 3 + g.rng.Intn(120)
		switch {
		case g.n%97 < 4: // a burst of trouble
			out = append(out, fmt.Sprintf(`{"time":"%s","level":"error","msg":"upstream timeout","service":"payments","path":"%s","attempt":%d,"err":"context deadline exceeded after 2000ms"}`, ts, path, g.n%3+1))
		case r < 55:
			out = append(out, fmt.Sprintf(`{"time":"%s","level":"info","msg":"request","method":"%s","path":"%s","status":200,"ms":%d,"bytes":%d}`, ts, method, path, ms, 200+g.rng.Intn(9000)))
		case r < 70:
			out = append(out, fmt.Sprintf(`{"time":"%s","level":"debug","msg":"cache lookup","key":"%s:%d","hit":%t}`, ts, strings.TrimPrefix(path, "/api/"), g.rng.Intn(500), g.rng.Intn(3) > 0))
		case r < 80:
			out = append(out, fmt.Sprintf(`%s WARN slow request %s %s took %dms (limit 100ms)`, ts, method, path, 100+g.rng.Intn(400)))
		case r < 90:
			out = append(out, fmt.Sprintf(`%s INFO worker %d picked up job %d`, ts, g.rng.Intn(8), 1000+g.n))
		case r < 95:
			out = append(out, fmt.Sprintf(`%s TRACE tick %d`, ts, g.n))
		case r < 98:
			out = append(out, fmt.Sprintf(`{"time":"%s","level":"warn","msg":"retrying","path":"%s","status":503,"retry_in_ms":%d}`, ts, path, 100*(1+g.rng.Intn(5))))
		default:
			out = append(out, fmt.Sprintf(`%s ERROR failed to write to disk: no space left on device`, ts))
		}
	}
	return out
}
