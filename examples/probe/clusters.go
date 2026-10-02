package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/term"
)

const dsrTimeout = 500 * time.Millisecond

type sample struct{ name, text string }

var samples = []sample{
	{"family ZWJ", "\U0001F468\u200D\U0001F469\u200D\U0001F467"},
	{"flag", "\U0001F1EF\U0001F1F5"},
	{"skin tone", "\U0001F44D\U0001F3FD"},
	{"VS16 heart", "\u2764\uFE0F"},
	{"combining accent", "e\u0301"},
}

// clusterResult is one sample: the columns the terminal advanced (measured
// only when answered) beside what ansi.Width says.
type clusterResult struct {
	name     string
	answered bool
	measured int
	width    int
}

// reader returns the next chunk of terminal input, or ok=false on timeout.
type reader func(timeout time.Duration) (data string, ok bool)

// parseCPR extracts the column from a cursor position report ESC [ row ; col R
// and reports whether s held one.
func parseCPR(s string) (col int, ok bool) {
	i := strings.Index(s, "\x1b[")
	if i < 0 {
		return 0, false
	}
	var row int
	if _, err := fmt.Sscanf(s[i:], "\x1b[%d;%dR", &row, &col); err != nil || col < 1 {
		return 0, false
	}
	return col, true
}

// measureClusters prints each sample at column 1, asks DSR 6 and records the
// advance. A sample whose reply does not arrive is marked unanswered; no value
// is guessed, and probing stops at the first silence.
func measureClusters(w io.Writer, read reader, timeout time.Duration) []clusterResult {
	res := make([]clusterResult, 0, len(samples))
	silent := false
	for _, s := range samples {
		r := clusterResult{name: s.name, width: ansi.Width(s.text)}
		if !silent {
			fmt.Fprint(w, "\r\x1b[2K"+s.text+"\x1b[6n")
			var buf string
			for {
				d, ok := read(timeout)
				if !ok {
					break
				}
				buf += d
				if col, found := parseCPR(buf); found {
					r.answered, r.measured = true, col-1
					break
				}
			}
			fmt.Fprint(w, "\r\x1b[2K")
			silent = !r.answered
		}
		res = append(res, r)
	}
	return res
}

// formatClusters renders the results one line each.
func formatClusters(res []clusterResult) string {
	var b strings.Builder
	for _, r := range res {
		if r.answered {
			fmt.Fprintf(&b, "  cluster %-17s measured=%d ansi.Width=%d\n", r.name, r.measured, r.width)
		} else {
			fmt.Fprintf(&b, "  cluster %-17s no answer to cursor position report (ansi.Width=%d)\n", r.name, r.width)
		}
	}
	return b.String()
}

// runClusterProbe measures on the real terminal, or reports why it cannot.
func runClusterProbe() string {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return "  clusters: not measured (stdin/stdout is not a terminal)\n"
	}
	st, err := term.MakeRaw(fd)
	if err != nil {
		return "  clusters: not measured (" + err.Error() + ")\n"
	}
	ch := make(chan string, 16)
	go func() { // may outlive a timeout; it holds at most one pending read
		b := make([]byte, 64)
		for {
			n, err := os.Stdin.Read(b)
			if n > 0 {
				ch <- string(b[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	read := func(t time.Duration) (string, bool) {
		select {
		case d := <-ch:
			return d, true
		case <-time.After(t):
			return "", false
		}
	}
	res := measureClusters(os.Stdout, read, dsrTimeout)
	_ = term.Restore(fd, st)
	return formatClusters(res)
}
