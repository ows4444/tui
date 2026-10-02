//go:build darwin

package ptytest

import (
	"os"
	"syscall"
	"unsafe"
)

// Open opens a new pseudo-terminal pair. The caller closes both files.
func Open() (master, slave *os.File, err error) {
	master, err = os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	fail := func(err error) (*os.File, *os.File, error) {
		_ = master.Close()
		return nil, nil, err
	}
	if err := ioctl(master.Fd(), syscall.TIOCPTYGRANT, nil); err != nil {
		return fail(err)
	}
	if err := ioctl(master.Fd(), syscall.TIOCPTYUNLK, nil); err != nil {
		return fail(err)
	}
	var name [128]byte
	if err := ioctl(master.Fd(), syscall.TIOCPTYGNAME, unsafe.Pointer(&name[0])); err != nil { // #nosec G103 -- ioctl requires unsafe.Pointer
		return fail(err)
	}
	n := 0
	for n < len(name) && name[n] != 0 {
		n++
	}
	slave, err = os.OpenFile(string(name[:n]), os.O_RDWR, 0)
	if err != nil {
		return fail(err)
	}
	return master, slave, nil
}

// Termios returns the terminal settings of f.
func Termios(f *os.File) (syscall.Termios, error) {
	var t syscall.Termios
	err := ioctlFile(f, syscall.TIOCGETA, unsafe.Pointer(&t)) // #nosec G103 -- ioctl requires unsafe.Pointer
	return t, err
}
