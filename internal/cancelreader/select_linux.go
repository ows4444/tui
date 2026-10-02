//go:build linux

package cancelreader

import "syscall"

// selectRead blocks until a descriptor in set is readable.
func selectRead(n int, set *syscall.FdSet) error {
	_, err := syscall.Select(n, set, nil, nil, nil)
	return err
}
