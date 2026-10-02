package render

import (
	"strings"
	"testing"
)

const (
	apc       = "\x1b_Gq=2;payload\x1b\\"
	kittyPlac = "\x1b_Ga=T,f=100,C=1,c=3,r=2,i=7,p=7,q=2,m=0;AAAA\x1b\\"
)

// draw feeds lines as the next frame of c.
func draw(t *testing.T, c *Cells, prevRows int, lines ...string) (string, Stats) {
	t.Helper()
	out, st, ok := c.Frame(Frame{
		Lines: lines, Max: max(len(lines), prevRows), PrevRows: prevRows, Width: 40, Height: 100,
		RegionTop: func() string { return "<top>" }, FitLine: func(s string) string { return s },
	})
	if !ok {
		t.Fatalf("frame %q fell back (%q)", lines, c.Reason())
	}
	return out, st
}

// When a View line contains an APC or DCS sequence, the cell renderer shall
// draw the frame without falling back (also PM and SOS).
func TestOpaqueSequencesDrawWithoutFallback(t *testing.T) {
	for name, seq := range map[string]string{
		"APC": "\x1b_Gq=2;payload\x1b\\",
		"DCS": "\x1bP1$r0m\x1b\\",
		"PM":  "\x1b^note\x1b\\",
		"SOS": "\x1bXnote\x1b\\",
	} {
		c := New()
		out, _ := draw(t, c, 0, "ab"+seq+"cd", "next")
		if c.Reason() != "" || !c.Valid() {
			t.Errorf("%s: reason %q valid %v", name, c.Reason(), c.Valid())
		}
		if !strings.Contains(out, seq) {
			t.Errorf("%s: output %q lacks the sequence", name, out)
		}
		// Zero width: the visible text is intact and the sequence sits between b and c.
		if i := strings.Index(out, "ab"); i < 0 || !strings.Contains(out[i:], "cd") {
			t.Errorf("%s: output %q", name, out)
		}
	}
	// An unterminated string is a row-level problem, not a frame-wide one.
	c := New()
	_, st := draw(t, c, 0, "ok", "a\x1b_Gnever ends")
	if !c.Valid() || len(st.Fallbacks) != 1 || st.Fallbacks[0] != (RowFallback{1, "unterminated_string_sequence"}) {
		t.Errorf("fallbacks = %+v", st.Fallbacks)
	}
}

// When a row holding an opaque segment is unchanged, the system shall not
// re-send the segment; when the row is rewritten, it is sent again.
func TestOpaqueSegmentNotResentForUnchangedRow(t *testing.T) {
	c := New()
	l0 := "x" + apc + "  y"
	draw(t, c, 0, l0, "one")
	out, st := draw(t, c, 2, l0, "two")
	if strings.Contains(out, apc) || st.Changed != 1 || !strings.Contains(out, "two") {
		t.Errorf("unchanged row re-sent its segment: %q (changed %d)", out, st.Changed)
	}
	// Rewriting the row re-sends it, at its anchor.
	out, _ = draw(t, c, 2, "X"+apc+"  y", "two")
	if strings.Count(out, apc) != 1 {
		t.Errorf("rewritten row: %q, want the segment once", out)
	}
	// A changed segment alone (same cells) is a change.
	out, st = draw(t, c, 2, "X"+"\x1b_Gq=2;other\x1b\\"+"  y", "two")
	if !strings.Contains(out, "other") || st.Changed != 1 {
		t.Errorf("changed segment not sent: %q (changed %d)", out, st.Changed)
	}
}

// The segment is anchored to its cell: it is re-sent at that column.
func TestOpaqueSegmentIsAnchoredToItsColumn(t *testing.T) {
	c := New()
	draw(t, c, 0, "abcdef"+apc)
	out, _ := draw(t, c, 1, "abcXef"+apc)
	// Cell 3 changes, then the cursor moves back to the anchor (column 6).
	if !strings.Contains(out, "X") || !strings.HasSuffix(strings.TrimSuffix(out, "\x1b[0m"), "\x1b[2C"+apc) {
		t.Errorf("out %q: segment not re-sent after a move to its anchor column", out)
	}
}

// When one row of a frame contains an unsupported sequence, the system shall
// draw every other row from the cell grid.
func TestUnsupportedRowFallsBackAloneAndOthersUseTheGrid(t *testing.T) {
	c := New()
	out, st := draw(t, c, 0, "aaa", "b\x01b", "ccc", "d\x1b[6md")
	if !c.Valid() || c.Reason() != "" {
		t.Fatalf("whole frame fell back: valid %v reason %q", c.Valid(), c.Reason())
	}
	want := []RowFallback{{1, "control_character"}, {3, "unsupported_SGR"}}
	if len(st.Fallbacks) != 2 || st.Fallbacks[0] != want[0] || st.Fallbacks[1] != want[1] {
		t.Fatalf("fallbacks = %+v, want %+v", st.Fallbacks, want)
	}
	if !strings.Contains(out, "\x1b[2Kb\x01b") {
		t.Errorf("bad row not drawn from its line: %q", out)
	}
	if !strings.Contains(out, "aaa") || !strings.Contains(out, "ccc") || c.GridRows() != 4 {
		t.Errorf("good rows missing: %q", out)
	}
	// Next frame: only a good row changes. The bad rows are not redrawn, and
	// the good row is a cell diff (one cell), not a line rewrite.
	out, st = draw(t, c, 4, "aaa", "b\x01b", "cXc", "d\x1b[6md")
	if len(st.Fallbacks) != 0 || strings.Contains(out, "\x01") || strings.Contains(out, "\x1b[2K") || !strings.Contains(out, "X") || strings.Contains(out, "ccc") {
		t.Errorf("second frame %q fallbacks %+v", out, st.Fallbacks)
	}
	// A bad row becoming good redraws that row from cells and clears it first.
	out, _ = draw(t, c, 4, "aaa", "bbb", "cXc", "d\x1b[6md")
	if !strings.Contains(out, "\x1b[K") || !strings.Contains(out, "bbb") {
		t.Errorf("recovered row %q", out)
	}
}

// A kitty placement that leaves the view or moves is deleted; one that stays
// is not, and is re-sent with its row.
func TestKittyPlacementDeletedWhenRemovedOrMoved(t *testing.T) {
	del := KittyDelete(7)
	if del != "\x1b_Ga=d,d=I,i=7,q=2\x1b\\" {
		t.Fatalf("KittyDelete = %q", del)
	}
	c := New()
	out, _ := draw(t, c, 0, kittyPlac+"   ", "   ", "text")
	if strings.Contains(out, del) {
		t.Errorf("first frame deletes: %q", out)
	}
	// Stays: row 0 unchanged, not re-sent, not deleted.
	out, _ = draw(t, c, 3, kittyPlac+"   ", "   ", "TEXT")
	if strings.Contains(out, del) || strings.Contains(out, kittyPlac) {
		t.Errorf("unchanged placement touched: %q", out)
	}
	// Moves to row 1: delete, then place at the new row.
	out, _ = draw(t, c, 3, "   ", kittyPlac+"   ", "TEXT")
	if i, j := strings.Index(out, del), strings.Index(out, kittyPlac); i < 0 || j < 0 || i > j {
		t.Errorf("moved placement: %q, want delete then place", out)
	}
	// Removed: delete, no placement.
	out, _ = draw(t, c, 3, "   ", "   ", "TEXT")
	if !strings.Contains(out, del) || strings.Contains(out, kittyPlac) {
		t.Errorf("removed placement: %q", out)
	}
	// Gone already: nothing more to delete.
	out, _ = draw(t, c, 3, "   ", "   ", "text")
	if strings.Contains(out, del) {
		t.Errorf("deleted twice: %q", out)
	}
}

// A segment that changes alone on a lower row is sent on that row.
func TestOpaqueSegmentChangeAloneMovesToItsRow(t *testing.T) {
	c := New()
	draw(t, c, 0, "top", "X"+apc)
	out, _ := draw(t, c, 2, "top", "X\x1b_Gq=2;other\x1b\\")
	if !strings.HasPrefix(out, "<top>\r\n") || !strings.Contains(out, "\x1b_Gq=2;other\x1b\\") {
		t.Errorf("out %q: segment not sent on row 1", out)
	}
}

func TestKittyPlacementID(t *testing.T) {
	for seq, want := range map[string]uint32{
		kittyPlac:                        7,
		"\x1b_Ga=p,i=9\x1b\\":            9,
		"\x1b_Ga=t,i=9;AA\x1b\\":         0, // transmit only: nothing placed
		"\x1b_Gm=0;AAAA\x1b\\":           0, // continuation chunk
		"\x1b_Ga=T,f=100;AA\x1b\\":       0, // no id
		"\x1b_Ga=d,d=I,i=7,q=2\x1b\\":    0,
		"\x1b_Xi=7\x1b\\":                0,
		"\x1b_Ga=T,i=99999999999;\x1b\\": 0,
	} {
		if got := kittyPlacementID(seq); got != want {
			t.Errorf("kittyPlacementID(%q) = %d, want %d", seq, got, want)
		}
	}
}
