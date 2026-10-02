package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/internal/cellcheck"
	"github.com/ows4444/tui/layout"
)

// When the dashboard is drawn with the cell renderer at every golden width and
// size, the frame log shall record zero fallbacks (spec S11).
func TestCellRendererNoFallbacks(t *testing.T) {
	if fb := cellcheck.Fallbacks(initialModel(), 400, 200); len(fb) > 0 {
		t.Errorf("natural view: %v", fb)
	}
	for _, w := range []int{40, 120} {
		s := initialModel().screen()
		size := s.Measure(layout.Loose(layout.Size{W: w, H: layout.Unbounded}))
		size.W = w
		view := s.Render(size)
		if fb := cellcheck.FallbacksView(view, w, strings.Count(view, "\n")+1); len(fb) > 0 {
			t.Errorf("width %d: %v", w, fb)
		}
	}
	for _, tc := range sizeGoldens {
		view := initialModel().screen().Render(tc.size)
		if fb := cellcheck.FallbacksView(view, tc.size.W, tc.size.H); len(fb) > 0 {
			t.Errorf("%v: %v", tc.size, fb)
		}
	}
}
