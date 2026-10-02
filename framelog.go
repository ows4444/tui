package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui/internal/render"
)

// frameStats is what renderBody (or renderAccessible) recorded about the frame
// it just built, for WithFrameLog.
type frameStats struct {
	rows, changed int
	full          bool // no previous frame to diff against
	accessible    bool
	fallback      string // WithCellRenderer frame drawn by the line renderer, and why
	// rowFallbacks are the rows of a cell-rendered frame drawn with the line
	// strategy, and why.
	rowFallbacks []render.RowFallback
}

// frameClock returns the time to measure a frame from, or the zero time when no
// frame log is set, so an app without WithFrameLog pays nothing for the timing.
func (p *Program) frameClock() time.Time {
	if p.frameLog == nil && p.inspectorKey == "" {
		return time.Time{}
	}
	return time.Now()
}

// logFrame writes the WithFrameLog line for the frame just written: bytes is
// how much it wrote and start is when frameClock said it began. It runs on the
// event loop goroutine, after the terminal write, and ignores write errors.
func (p *Program) logFrame(bytes int, start time.Time) {
	p.noteFrame(bytes, start)
	if p.frameLog == nil {
		return
	}
	p.frameN++
	kind := "diff"
	switch {
	case p.frameStats.accessible:
		kind = "accessible"
	case p.frameStats.fallback != "":
		kind = "fallback"
	case p.frameStats.full:
		kind = "full"
	}
	reason := ""
	if kind == "fallback" {
		reason = " reason=" + p.frameStats.fallback
	}
	if len(p.frameStats.rowFallbacks) > 0 {
		var b strings.Builder
		b.WriteString(" fallback_rows=")
		for i, f := range p.frameStats.rowFallbacks {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.Itoa(f.Row) + ":" + f.Reason)
		}
		reason += b.String()
	}
	var since time.Duration
	if !p.loopStart.IsZero() {
		since = time.Since(p.loopStart)
	}
	_, _ = fmt.Fprintf(p.frameLog, "n=%d t=%d kind=%s rows=%d changed=%d bytes=%d us=%d%s\n",
		p.frameN, since.Milliseconds(), kind, p.frameStats.rows, p.frameStats.changed, bytes, time.Since(start).Microseconds(), reason)
}
