//go:build !darwin && !linux

package termpane

import (
	"errors"
	"os"
	"os/exec"
)

// errUnsupported is the reason a pane cannot start where there are no
// pseudo-terminals.
var errUnsupported = errors.New("termpane: pseudo-terminals are not supported on this system")

func startProcess(*exec.Cmd, int, int) (*os.File, error) { return nil, errUnsupported }

func resizeTerminal(*os.File, int, int) error { return errUnsupported }
