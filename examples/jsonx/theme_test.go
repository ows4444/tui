package main

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/uitest"
	"strings"
)

// TestFollowsTheThemeAndSpeaksPlainly checks that the model draws in whatever theme
// the Program gives it and hands a screen reader plain text.
func TestFollowsTheThemeAndSpeaksPlainly(t *testing.T) {
	uitest.Check(t, func() tui.Model {
		root, _ := parse(strings.NewReader(sampleJSON))
		return newModel(root, "sample.json")
	})
}
