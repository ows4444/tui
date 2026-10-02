//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package cancelreader

import (
	"syscall"
	"testing"
)

func TestFdSetBitHelpers(t *testing.T) {
	if fdSetSize < 256 || fdSetSize%32 != 0 {
		t.Fatalf("fdSetSize = %d, want the bit capacity of the platform FdSet", fdSetSize)
	}
	var set syscall.FdSet
	for _, fd := range []int{0, 1, 31, 32, 63, 64, fdSetSize - 1} {
		if fdIsSet(&set, fd) {
			t.Errorf("fd %d set in an empty set", fd)
		}
		fdSetAdd(&set, fd)
		if !fdIsSet(&set, fd) {
			t.Errorf("fd %d not set after fdSetAdd", fd)
		}
	}
	if fdIsSet(&set, 2) || fdIsSet(&set, 33) {
		t.Error("a bit that was never added is set")
	}
}

func TestBitHelpersForEveryWordType(t *testing.T) {
	check := func(name string, set func(fd int), has func(fd int) bool, bits int) {
		for _, fd := range []int{0, 1, bits - 1, bits, bits + 1, 3*bits - 1} {
			set(fd)
			if !has(fd) {
				t.Errorf("%s: bit %d lost", name, fd)
			}
		}
		if has(2) {
			t.Errorf("%s: bit 2 set by accident", name)
		}
	}
	w32, w64 := make([]uint32, 4), make([]uint64, 4)
	i32, i64 := make([]int32, 4), make([]int64, 4)
	check("uint32", func(fd int) { setBit(w32, fd) }, func(fd int) bool { return hasBit(w32, fd) }, 32)
	check("uint64", func(fd int) { setBit(w64, fd) }, func(fd int) bool { return hasBit(w64, fd) }, 64)
	check("int32", func(fd int) { setBit(i32, fd) }, func(fd int) bool { return hasBit(i32, fd) }, 32)
	check("int64", func(fd int) { setBit(i64, fd) }, func(fd int) bool { return hasBit(i64, fd) }, 64)
	if wordBits(w32) != 32 || wordBits(i64) != 64 {
		t.Error("wordBits wrong")
	}
}
