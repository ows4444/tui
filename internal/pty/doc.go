// Package pty opens pseudo-terminals with /dev/ptmx and the pty ioctls
// directly, with no dependencies, for the termpane widget. It works on darwin
// and linux; elsewhere Open reports ErrUnsupported. It is internal: not public API.
package pty
