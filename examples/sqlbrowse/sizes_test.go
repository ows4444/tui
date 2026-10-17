package main

import (
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/testutil"
)

// stillClock stands still, so a query always takes 0 and the screen is the
// same every time.
func stillClock() time.Time { return time.Date(2026, 10, 18, 9, 30, 0, 0, time.UTC) }

func freshModel() model { return newModel(fixture(), "", stillClock) }

// TestSizeMatrix renders the example at 40x10, 80x24, 120x40 and 300x80 and
// checks each against its golden file with no line wider than the terminal.
func TestSizeMatrix(t *testing.T) {
	testutil.SizeMatrix(t, func() tui.Model { return freshModel() })
}
