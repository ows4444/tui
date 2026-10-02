//go:build linux

package ptytest

import (
	"os"
	"strconv"
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
		_ = master.Close() // best effort: the original error is the one to report
		return nil, nil, err
	}
	var unlock int32                                                                        // 0 unlocks the slave
	if err := ioctl(master.Fd(), syscall.TIOCSPTLCK, unsafe.Pointer(&unlock)); err != nil { // #nosec G103 -- ioctl requires unsafe.Pointer
		return fail(err)
	}
	var n uint32
	if err := ioctl(master.Fd(), syscall.TIOCGPTN, unsafe.Pointer(&n)); err != nil { // #nosec G103 -- ioctl requires unsafe.Pointer
		return fail(err)
	}
	slave, err = os.OpenFile("/dev/pts/"+strconv.FormatUint(uint64(n), 10), os.O_RDWR, 0)
	if err != nil {
		return fail(err)
	}
	return master, slave, nil
}

// Termios returns the terminal settings of f.
func Termios(f *os.File) (syscall.Termios, error) {
	var t syscall.Termios
	err := ioctlFile(f, syscall.TCGETS, unsafe.Pointer(&t)) // #nosec G103 -- ioctl requires unsafe.Pointer
	return t, err
}
