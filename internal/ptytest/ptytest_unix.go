//go:build darwin || linux

package ptytest

import (
	"os"
	"syscall"
	"unsafe"
)

func ioctl(fd uintptr, req uintptr, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

// ioctlFile runs an ioctl on f without calling f.Fd(), which would switch
// the file back to blocking mode under the caller.
func ioctlFile(f *os.File, req uintptr, arg unsafe.Pointer) error {
	rc, err := f.SyscallConn()
	if err != nil {
		return err
	}
	var ioerr error
	if err := rc.Control(func(fd uintptr) { ioerr = ioctl(fd, req, arg) }); err != nil {
		return err
	}
	return ioerr
}

// SetSize sets the window size of the terminal that f (either end of a
// pty) is attached to.
func SetSize(f *os.File, cols, rows uint16) error {
	ws := struct{ Row, Col, Xpixel, Ypixel uint16 }{Row: rows, Col: cols}
	return ioctlFile(f, syscall.TIOCSWINSZ, unsafe.Pointer(&ws)) // #nosec G103 -- ioctl requires unsafe.Pointer
}
