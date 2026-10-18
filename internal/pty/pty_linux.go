//go:build linux

package pty

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

// The device paths, variables so that a test can point at something that is not
// a pseudo-terminal and reach the error returns.
var (
	ptmxPath = "/dev/ptmx"
	ptsDir   = "/dev/pts/"
)

// Open opens a new pseudo-terminal pair. The caller closes both files.
func Open() (master, slave *os.File, err error) {
	master, err = os.OpenFile(ptmxPath, os.O_RDWR, 0)
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
	slave, err = os.OpenFile(ptsDir+strconv.FormatUint(uint64(n), 10), os.O_RDWR, 0)
	if err != nil {
		return fail(err)
	}
	return master, slave, nil
}
