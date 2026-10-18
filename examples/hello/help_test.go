package main

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/uitest"
)

// TestHelpOpensAndAnyKeyClosesIt checks the help screen.
func TestHelpOpensAndAnyKeyClosesIt(t *testing.T) {
	uitest.CheckHelp(t, func() tui.Model { return initialModel() }, tui.Key{Type: tui.KeyF1}, "move focus between the field")
}
