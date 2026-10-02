//go:build freebsd

package cancelreader

import "syscall"

// FreeBSD names the bit array X__fds_bits.
func fdSetAdd(s *syscall.FdSet, fd int)     { setBit(s.X__fds_bits[:], fd) }
func fdIsSet(s *syscall.FdSet, fd int) bool { return hasBit(s.X__fds_bits[:], fd) }
func fdSetBits(s *syscall.FdSet) int        { return len(s.X__fds_bits) * wordBits(s.X__fds_bits[:]) }
