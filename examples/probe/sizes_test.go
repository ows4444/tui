package main

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/testutil"
)

// TestSizeMatrix renders the example at 40x10, 80x24, 120x40 and 300x80 and
// checks each against its golden file with no line wider than the terminal.
func TestSizeMatrix(t *testing.T) {
	// View prints these; pin them so the goldens do not depend on the machine.
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("COLORTERM", "truecolor")
	t.Setenv("NO_COLOR", "")
	// The report is inline, so it is allowed to be taller than the window.
	testutil.SizeMatrix(t, func() tui.Model { return newModel(ansi.TrueColor) }, testutil.NoHeightCheck)
}
