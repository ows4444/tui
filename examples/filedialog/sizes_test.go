package main

import (
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/testutil"
)

// TestSizeMatrix renders the dialog at 40x10, 80x24, 120x40 and 300x80 and
// checks each against its golden file with no line wider than the terminal. The
// folder is a fixture with fixed times, and the path bar starts at it, so the
// goldens do not hold the temporary path.
func TestSizeMatrix(t *testing.T) {
	root := fixture(t)
	testutil.SizeMatrix(t, func() tui.Model {
		m := newModel(root, false)
		m.root, m.loc = root, time.UTC
		return m
	})
}
