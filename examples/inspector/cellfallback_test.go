package main

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/cellcheck"
)

// When the inspector screens are drawn with the cell renderer, the frame log
// shall record zero fallbacks (spec S11).
func TestCellRendererNoFallbacks(t *testing.T) {
	m := send(initialModel(), tui.Key{Type: tui.KeyRight})
	m = send(m, tui.Key{Type: tui.KeyEnter})
	if fb := cellcheck.Fallbacks(initialModel(), 200, 100); len(fb) > 0 {
		t.Errorf("initial: %v", fb)
	}
	if fb := cellcheck.Fallbacks(m, 200, 100); len(fb) > 0 {
		t.Errorf("json tab: %v", fb)
	}
}
