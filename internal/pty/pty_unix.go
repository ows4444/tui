//go:build darwin || linux

package pty

import (
	"os"
	"syscall"
	"unsafe"
)

// Supported reports whether pseudo-terminals can be opened on this system.
const Supported = true

func ioctl(fd uintptr, req uintptr, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

// SetSize sets the window size of the terminal that f (either end of a pty) is
// attached to. It does not call f.Fd(), which would put f back in blocking mode.
func SetSize(f *os.File, cols, rows uint16) error {
	ws := struct{ Row, Col, Xpixel, Ypixel uint16 }{Row: rows, Col: cols}
	rc, err := f.SyscallConn()
	if err != nil {
		return err
	}
	var ioerr error
	if err := rc.Control(func(fd uintptr) {
		ioerr = ioctl(fd, syscall.TIOCSWINSZ, unsafe.Pointer(&ws)) // #nosec G103 -- ioctl requires unsafe.Pointer
	}); err != nil {
		return err
	}
	return ioerr
}
