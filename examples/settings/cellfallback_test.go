package main

import (
	"testing"

	"github.com/ows4444/tui/internal/cellcheck"
)

// When both settings tabs are drawn with the cell renderer, the frame log
// shall record zero fallbacks (spec S11).
func TestCellRendererNoFallbacks(t *testing.T) {
	for active, name := range []string{"general", "advanced"} {
		m := initialModel()
		m.tabs.SetActive(active)
		if fb := cellcheck.Fallbacks(m, 200, 100); len(fb) > 0 {
			t.Errorf("%s: %v", name, fb)
		}
	}
}
