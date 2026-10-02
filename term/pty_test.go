//go:build darwin || linux

package term

import (
	"os"

	"github.com/ows4444/tui/internal/ptytest"
)

// openPTY opens a pseudo-terminal pair for the tests in this package.
func openPTY() (master, slave *os.File, err error) { return ptytest.Open() }
