//go:build windows

package term

import (
	"syscall"
	"unsafe"
)

// The Windows syscall package doesn't wrap the Console API the way it
// wraps POSIX-style calls, so these are bound directly from kernel32.dll —
// this is the windows equivalent of the ioctl_darwin.go/ioctl_linux.go
// syscall.Syscall(SYS_IOCTL, ...) calls, just via a different calling
// convention.
var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

func getConsoleMode(fd int) (uint32, error) {
	var mode uint32
	r, _, err := procGetConsoleMode.Call(uintptr(fd), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return 0, err
	}
	return mode, nil
}

func setConsoleMode(fd int, mode uint32) error {
	r, _, err := procSetConsoleMode.Call(uintptr(fd), uintptr(mode))
	if r == 0 {
		return err
	}
	return nil
}
