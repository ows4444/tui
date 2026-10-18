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

	root := fixture(t)
	uitest.Check(t, func() tui.Model {
		m := newModel(root, false)
		m.root, m.loc = root, time.UTC
		return m
	})
}
