//go:build darwin || linux

package cancelreader

import (
	"os"

	"github.com/ows4444/tui/term"
)

// ptyRaw puts the terminal on f into raw mode (input arrives without a
// newline). The state is not restored: the test discards the pty.
func ptyRaw(f *os.File) (*term.State, error) {
	fd := 0
	rc, err := f.SyscallConn()
	if err != nil {
		return nil, err
	}
	if err := rc.Control(func(u uintptr) { fd = int(u) }); err != nil {
		return nil, err
	}
	return term.MakeRaw(fd)
}

// mustFd returns f's descriptor without changing its blocking mode.
func mustFd(f *os.File) uintptr {
	var fd uintptr
	rc, err := f.SyscallConn()
	if err != nil {
		panic(err)
	}
	if err := rc.Control(func(u uintptr) { fd = u }); err != nil {
		panic(err)
	}
	return fd
}
