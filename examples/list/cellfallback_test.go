package main

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/cellcheck"
	"github.com/ows4444/tui/multiselect"
)

// When the list screens are drawn with the cell renderer, the frame log shall
// record zero fallbacks (spec S11).
func TestCellRendererNoFallbacks(t *testing.T) {
	next, _ := initialModel().Update(tui.Key{Type: tui.KeySpace})
	if fb := cellcheck.Fallbacks(next, 200, 100); len(fb) > 0 {
		t.Errorf("selected: %v", fb)
	}
	next, _ = next.Update(multiselect.ConfirmedMsg{})
	if fb := cellcheck.Fallbacks(next, 200, 100); len(fb) > 0 {
		t.Errorf("confirmed: %v", fb)
	}
}
