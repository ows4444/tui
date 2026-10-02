//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package cancelreader

import "syscall"

// selectRead blocks until a descriptor in set is readable. On the BSDs and
// macOS syscall.Select returns only an error.
func selectRead(n int, set *syscall.FdSet) error {
	return syscall.Select(n, set, nil, nil, nil)
}
