package main

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/uitest"
	"time"
)

// TestFollowsTheThemeAndSpeaksPlainly checks that the model draws in whatever theme
// the Program gives it and hands a screen reader plain text.
func TestFollowsTheThemeAndSpeaksPlainly(t *testing.T) {
	uitest.Check(t, func() tui.Model {
		m := newModel()
		m.now = func() time.Time { return time.Unix(0, 0) }
		return m
	})
}
