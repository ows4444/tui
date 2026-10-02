package tui

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/internal/render"
)

func inputFor(prev, lines []string) frameInput {
	rows := max(len(prev), len(lines))
	return frameInput{lines: lines, rows: rows, prev: prev, regionTop: func() string { return "<top>" }, fitLine: func(s string) string { return s }}
}

// The line renderer rewrites only rows that differ and reports how many.
func TestLineFramesRewritesOnlyChangedRows(t *testing.T) {
	out, st, ok := lineFrames{}.draw(inputFor([]string{"a", "b", "c"}, []string{"a", "X", "c"}))
	if !ok {
		t.Fatal("the line renderer must always draw")
	}
	if !strings.HasPrefix(out, "<top>") || strings.Count(out, "X") != 1 || strings.Contains(out, "a") || strings.Contains(out, "c") {
		t.Errorf("out = %q, want only the changed row after the region top", out)
	}
	if st.rows != 3 || st.changed != 1 || st.full {
		t.Errorf("stats = %+v, want 3 rows, 1 changed, not full", st)
	}
}

func TestLineFramesFirstFrameIsFullAndShrinkBlanksRows(t *testing.T) {
	_, st, _ := lineFrames{}.draw(inputFor(nil, []string{"a", "b"}))
	if !st.full || st.changed != 2 {
		t.Errorf("first frame stats = %+v, want full with 2 changed", st)
	}
	out, st, _ := lineFrames{}.draw(inputFor([]string{"a", "b"}, []string{"a"}))
	if st.rows != 2 || st.changed != 1 || !strings.Contains(out, "\r\n") {
		t.Errorf("shrink: out %q stats %+v, want the vanished row blanked", out, st)
	}
}

// With overflow committed above, the commit is written instead of the region
// top and the frame counts as full.
func TestLineFramesWritesTheCommitInsteadOfTheRegionTop(t *testing.T) {
	in := inputFor(nil, []string{"live"})
	in.commit = "history\r\n"
	out, st, _ := lineFrames{}.draw(in)
	if !strings.HasPrefix(out, "history\r\n") || strings.Contains(out, "<top>") || !st.full {
		t.Errorf("out %q stats %+v", out, st)
	}
}

// The cell renderer declines what it cannot draw, and says why.
func TestCellFramesDeclinesAndReports(t *testing.T) {
	r := cellFrames{render.New()}
	in := inputFor(nil, []string{"a", "b"})
	in.height = 1 // a view taller than the terminal is a frame-wide problem
	if _, _, ok := r.draw(in); ok {
		t.Fatal("cell renderer drew a view taller than the terminal")
	}
	if r.reason() == "" {
		t.Error("declined without a reason")
	}
	good := inputFor(nil, []string{"ok"})
	if _, _, ok := r.draw(good); !ok || r.reason() != "" {
		t.Errorf("clean frame: ok=%v reason=%q", ok, r.reason())
	}
	r.reset()
	if r.c.Valid() {
		t.Error("reset left a previous frame held")
	}
}
