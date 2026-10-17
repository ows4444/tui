package render

import (
	"fmt"
	"strings"
	"testing"
)

// manyStyles is one line of n cells, each in a different foreground colour.
func manyStyles(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "\x1b[38;2;%d;%d;0mx", i%256, i/256)
	}
	return b.String()
}

// TestWholeFrameFallbackCases lists every reason the cell renderer gives a whole
// frame to the line renderer. Anything else a row can do wrong costs that row
// only (TestCovParseRowCases lists those reasons).
func TestWholeFrameFallbackCases(t *testing.T) {
	for _, c := range []struct {
		name   string
		height int
		lines  []string
		reason string
	}{
		{"a view taller than the terminal", 2, []string{"a", "b", "c"}, "view_taller_than_terminal"},
		{"more distinct styles than the table holds", 0, []string{manyStyles(maxCellStyles + 10)}, "style_table_full"},
	} {
		t.Run(c.name, func(t *testing.T) {
			cells := New()
			if _, _, ok := covFrame(cells, 0, 80, c.height, c.lines...); ok || cells.Reason() != c.reason {
				t.Fatalf("ok %v, reason %q; want a whole-frame fallback with %q", ok, cells.Reason(), c.reason)
			}
			// The next frame is drawn by the cell renderer again.
			if out, _, ok := covFrame(cells, 0, 80, 0, "plain"); !ok || !strings.Contains(out, "plain") {
				t.Fatalf("the frame after a fallback: ok %v, %q", ok, out)
			}
		})
	}
}

func TestRowFallbacksDoNotTakeTheFrame(t *testing.T) {
	cells := New()
	out, st, ok := covFrame(cells, 0, 80, 0, "fine", "bad\x01row", "also fine")
	if !ok {
		t.Fatalf("one bad row took the whole frame: %q", cells.Reason())
	}
	if len(st.Fallbacks) != 1 || st.Fallbacks[0].Row != 1 || st.Fallbacks[0].Reason != "control_character" {
		t.Fatalf("Fallbacks = %+v", st.Fallbacks)
	}
	for _, want := range []string{"fine", "also fine"} {
		if !strings.Contains(out, want) {
			t.Errorf("the output lost %q: %q", want, out)
		}
	}
}
