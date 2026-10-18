package main

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/testutil"
	"github.com/ows4444/tui/motion"
)

// TestSizeMatrix renders the example at 40x10, 80x24, 120x40 and 300x80 and
// checks each against its golden file with no line wider than the terminal. The
// motion preference is pinned so the goldens do not depend on the environment.
func TestSizeMatrix(t *testing.T) {
	testutil.SizeMatrix(t, func() tui.Model {
		m := newModel()
		m.pref, m.still = motion.Normal, false
		return m
	})
}
