package cellbuf_test

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/layout"
)

func TestColourConstructorsWrapAndRender(t *testing.T) {
	if cellbuf.Basic(9) != cellbuf.Basic(1) || cellbuf.Bright(9) != cellbuf.Bright(1) {
		t.Error("Basic and Bright take n modulo 8")
	}
	if cellbuf.Indexed(257) != cellbuf.Indexed(1) {
		t.Error("Indexed takes n modulo 256")
	}
	if cellbuf.Bright(1) == cellbuf.Basic(1) || cellbuf.Indexed(1) == cellbuf.Basic(1) {
		t.Error("colour kinds must differ")
	}
	b := cellbuf.New(3, 1)
	b.SetString(0, 0, "a", b.StyleID(cellbuf.Style{FG: cellbuf.Bright(2)}))
	b.SetString(1, 0, "b", b.StyleID(cellbuf.Style{FG: cellbuf.Indexed(200)}))
	got := b.String()
	if !strings.Contains(got, "92") || !strings.Contains(got, "38;5;200") {
		t.Errorf("bright and indexed colours not rendered: %q", got)
	}
}

func TestStyleOfUnknownIDIsDefault(t *testing.T) {
	b := cellbuf.New(1, 1)
	if got := b.Style(cellbuf.StyleID(60000)); got != (cellbuf.Style{}) {
		t.Errorf("Style of an unknown id = %+v, want the zero Style", got)
	}
}

func TestSetMeasurerChangesClusterWidths(t *testing.T) {
	family := "👨‍👩‍👧" // one cluster of three emoji joined by ZWJ
	on, off := cellbuf.New(10, 1), cellbuf.New(10, 1)
	on.SetMeasurer(ansi.ClusterMeasurer(true))
	off.SetMeasurer(ansi.ClusterMeasurer(false))
	nOn := on.SetString(0, 0, family, 0)
	nOff := off.SetString(0, 0, family, 0)
	if nOn != 2 {
		t.Errorf("cluster measurer wrote %d columns, want 2", nOn)
	}
	if nOff <= nOn {
		t.Errorf("per-rune measurer wrote %d columns, want more than %d", nOff, nOn)
	}
}

func TestSurfaceClipPutRepeat(t *testing.T) {
	b := cellbuf.New(10, 3)
	s := cellbuf.Layout(b)
	if got := s.Clip(); got != (layout.Rect{W: 10, H: 3}) {
		t.Fatalf("initial Clip = %+v", got)
	}
	s.SetClip(layout.Rect{X: -5, Y: 1, W: 100, H: 100}) // clipped to the buffer
	if got := s.Clip(); got != (layout.Rect{X: 0, Y: 1, W: 10, H: 2}) {
		t.Errorf("clipped Clip = %+v", got)
	}
	s.SetClip(layout.Rect{X: 2, Y: 0, W: 4, H: 3})
	if n := s.Put(2, 0, "abcdefgh"); n != 4 {
		t.Errorf("Put wrote %d columns, want 4 (clipped)", n)
	}
	if n := s.Put(2, 0, ""); n != 0 {
		t.Errorf("Put of \"\" wrote %d", n)
	}
	if n := s.Put(2, 1, "│"); n != 1 { // one non-ASCII cluster
		t.Errorf("Put of a box glyph wrote %d", n)
	}
	if n := s.Put(2, 2, "\x1b[1mhi\x1b[0m"); n != 2 { // styled
		t.Errorf("styled Put wrote %d", n)
	}
	s.Repeat(2, 2, 10, "-")
	s.Repeat(3, 1, 2, "═")
	s.Repeat(2, 1, 0, "x")  // n <= 0: nothing
	s.Repeat(2, 99, 3, "x") // row outside the clip: nothing
	lines := b.Lines()
	if !strings.HasPrefix(lines[0], "  abcd") || strings.Contains(lines[0], "e") {
		t.Errorf("row 0 = %q", lines[0])
	}
	if !strings.Contains(lines[1], "│══") {
		t.Errorf("row 1 = %q", lines[1])
	}
	if !strings.Contains(lines[2], "----") || strings.Count(lines[2], "-") != 4 {
		t.Errorf("row 2 = %q (Repeat must stop at the clip's right edge)", lines[2])
	}
	s.SetClip(layout.Rect{X: 50, Y: 50, W: 1, H: 1}) // outside: empty clip
	if got := s.Clip(); got != (layout.Rect{}) {
		t.Errorf("empty Clip = %+v", got)
	}
	if n := s.Put(0, 0, "zzz"); n != 0 {
		t.Errorf("Put under an empty clip wrote %d", n)
	}
}

func TestSurfaceRepeatLeftOfViewAndMultiRune(t *testing.T) {
	b := cellbuf.New(6, 1)
	s := cellbuf.Layout(b)
	s.Repeat(-2, 0, 5, "ab") // a two-rune cluster string repeats its first cluster
	s.Repeat(0, 0, 1, "")    // empty cluster falls back to a space
	if got := b.Lines()[0]; !strings.HasPrefix(got, " aa") || strings.Contains(got, "b") {
		t.Errorf("row = %q, want \" aa\" then blanks", got)
	}
}
