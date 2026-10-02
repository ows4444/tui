package main

import (
	"testing"

	"github.com/ows4444/tui/internal/cellcheck"
)

// When every login screen is drawn with the cell renderer, the frame log
// shall record zero fallbacks.
func TestCellRendererNoFallbacks(t *testing.T) {
	filled := initialModel()
	filled.design.pick[setFill] = fillSurface
	filled.design.pick[setTitle] = titleBorder
	filled.restyle()
	for name, m := range map[string]model{
		"empty":     initialModel(),
		"error":     signIn(initialModel(), demoUser, "nope"),
		"signed-in": signIn(initialModel(), demoUser, demoPassword),
		"designing": toggle(initialModel()),
		"filled":    filled,
	} {
		if fb := cellcheck.Fallbacks(m, 200, 100); len(fb) > 0 {
			t.Errorf("%s: %v", name, fb)
		}
	}
}
