package main

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/testutil"
)

// TestSizeMatrix renders the example at 40x10, 80x24, 120x40 and 300x80 and
// checks each against its golden file with no line wider than the terminal. It
// is an inline program, so its frame is not clipped to the window height.
func TestSizeMatrix(t *testing.T) {
	testutil.SizeMatrix(t, func() tui.Model {
		m := newModel(3, "")
		m.still = false
		return m
	}, testutil.NoHeightCheck)
}
