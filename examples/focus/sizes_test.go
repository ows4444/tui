package main

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/testutil"
)

// TestSizeMatrix renders the example at 40x10, 80x24, 120x40 and 300x80 and
// checks each against its golden file with no line wider than the terminal.
func TestSizeMatrix(t *testing.T) {
	testutil.SizeMatrix(t, func() tui.Model { return initialModel() })
}
