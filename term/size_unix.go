//go:build linux || darwin || dragonfly || freebsd || netbsd || openbsd

package term

import (
	"syscall"
	"unsafe"
)

type winsize struct {
	Row, Col, Xpixel, Ypixel uint16
}

// GetSize returns the terminal's width and height in columns/rows.
func GetSize(fd int) (width, height int, err error) {
	ws := &winsize{}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(ws))) // #nosec G103 -- ioctl requires unsafe.Pointer
	if errno != 0 {
		return 0, 0, errno
	}
	return int(ws.Col), int(ws.Row), nil
}
