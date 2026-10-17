//go:build darwin || linux

package termpane

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/ows4444/tui/internal/pty"
)

// startProcess runs cmd on a new pseudo-terminal of the given size and returns
// the master end. The child leads a session of its own with the terminal as its
// controlling one, so job control and Ctrl+C work in it.
func startProcess(cmd *exec.Cmd, cols, rows int) (*os.File, error) {
	master, slave, err := pty.Open()
	if err != nil {
		return nil, err
	}
	if err := pty.SetSize(master, uint16(cols), uint16(rows)); err != nil { // #nosec G115 -- sizes are clamped to 1..65535 by the caller
		_ = master.Close()
		_ = slave.Close()
		return nil, err
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
	err = cmd.Start()
	_ = slave.Close() // the child holds its own copy
	if err != nil {
		_ = master.Close()
		return nil, err
	}
	return master, nil
}

// resizeTerminal tells the child's terminal its new size.
func resizeTerminal(master *os.File, cols, rows int) error {
	return pty.SetSize(master, uint16(cols), uint16(rows)) // #nosec G115 -- sizes are clamped to 1..65535 by the caller
}
