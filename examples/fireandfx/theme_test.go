package main

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/uitest"
	"github.com/ows4444/tui/motion"
)

// TestFollowsTheThemeAndSpeaksPlainly checks that the model draws in whatever theme
// the Program gives it and hands a screen reader plain text.
func TestFollowsTheThemeAndSpeaksPlainly(t *testing.T) {
	uitest.Check(t, func() tui.Model {
		m := newModel()
		m.pref, m.still = motion.Normal, false
		return m
	})
}
