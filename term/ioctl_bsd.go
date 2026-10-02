//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package term

import (
	"syscall"
	"unsafe"
)

func ioctl(fd int, req uintptr, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

func ioctlGetTermios(fd int, t *syscall.Termios) error {
	return ioctl(fd, syscall.TIOCGETA, unsafe.Pointer(t)) // #nosec G103 -- ioctl requires unsafe.Pointer
}

func ioctlSetTermios(fd int, t *syscall.Termios) error {
	return ioctl(fd, syscall.TIOCSETA, unsafe.Pointer(t)) // #nosec G103 -- ioctl requires unsafe.Pointer
}
