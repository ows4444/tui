//go:build darwin || dragonfly || linux || netbsd || openbsd

package cancelreader

import "syscall"

func fdSetAdd(s *syscall.FdSet, fd int)     { setBit(s.Bits[:], fd) }
func fdIsSet(s *syscall.FdSet, fd int) bool { return hasBit(s.Bits[:], fd) }
func fdSetBits(s *syscall.FdSet) int        { return len(s.Bits) * wordBits(s.Bits[:]) }
