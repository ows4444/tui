package tui

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/render"
)

// frameRenderer turns the view's lines into the bytes that update the live
// region from the previous frame. Two implementations exist: cellFrames, the
// default, which diffs a grid of cells, and lineFrames, which diffs whole
// lines and can draw anything, so it is also the fallback when the cell
// renderer declines a frame. Program owns the shared bookkeeping (lastFrame,
// liveLines, committed); a frameRenderer only reads what it is given.
type frameRenderer interface {
	// draw returns the update for in. ok is false when this renderer cannot
	// draw the frame, in which case it has recorded nothing and reason says
	// why; the caller then uses another renderer.
	draw(in frameInput) (out string, st frameStats, ok bool)
	// reason is why the last draw declined, "" when it did not.
	reason() string
	// reset drops any state kept for diffing, so the next draw is a full one.
	reset()
}

// frameInput is one frame as the renderers see it.
type frameInput struct {
	lines   []string // the visible view lines, already fitted unless lazyFit
	rows    int      // rows to draw: max(len(lines), rows of the previous frame)
	lazyFit bool     // lines are not yet fitted to width; the renderer fits what it must
	width   int
	height  int      // terminal height; <= 0 when unknown
	prev    []string // the previous frame's lines, nil when there is none
	// commit is the bytes that scroll overflowing rows into history, written
	// before the live region is redrawn below them; "" when nothing overflowed.
	commit    string
	regionTop func() string // the sequence that returns the cursor to the region's first row
	fitLine   func(string) string
	measurer  ansi.Measurer // how this Program's terminal measures grapheme clusters
}

// lineFrames is the line renderer: it rewrites every line that differs from the
// previous frame. It keeps no state.
type lineFrames struct{}

func (lineFrames) reason() string { return "" }
func (lineFrames) reset()         {}

func (lineFrames) draw(in frameInput) (string, frameStats, bool) {
	var buf strings.Builder
	if in.commit != "" {
		buf.WriteString(in.commit)
	} else {
		buf.WriteString(in.regionTop())
	}
	changed := 0
	for i := 0; i < in.rows; i++ {
		var newLine string
		if i < len(in.lines) {
			newLine = in.lines[i]
		}
		haveOld := i < len(in.prev)
		if !haveOld || in.prev[i] != newLine {
			buf.WriteString(ansi.ClearLine)
			buf.WriteString(newLine)
			changed++
		}
		if i < in.rows-1 {
			buf.WriteString("\r\n")
		}
	}
	return buf.String(), frameStats{rows: in.rows, changed: changed, full: len(in.prev) == 0 || in.commit != ""}, true
}

// cellFrames is the cell renderer, a thin adapter over render.Cells.
type cellFrames struct{ c *render.Cells }

func (r cellFrames) reason() string { return r.c.Reason() }
func (r cellFrames) reset()         { r.c.Invalidate() }

func (r cellFrames) draw(in frameInput) (string, frameStats, bool) {
	out, st, ok := r.c.Frame(render.Frame{
		Lines: in.lines, Max: in.rows, LazyFit: in.lazyFit,
		Width: in.width, Height: in.height, PrevRows: len(in.prev),
		RegionTop: in.regionTop, FitLine: in.fitLine, Measurer: in.measurer,
	})
	if !ok {
		return "", frameStats{}, false
	}
	fs := frameStats{rows: st.Rows, changed: st.Changed, full: st.Full}
	if len(st.Fallbacks) > 0 {
		fs.rowFallbacks = append([]render.RowFallback(nil), st.Fallbacks...)
	}
	return out, fs, true
}

var (
	_ frameRenderer = lineFrames{}
	_ frameRenderer = cellFrames{}
)
