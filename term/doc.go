// Package term puts the controlling terminal into raw mode and reads its
// size. On darwin/linux this is done with raw ioctl syscalls (no
// golang.org/x/term); on windows it's done with the Console API (no
// golang.org/x/sys/windows). Every other file in this package is
// platform-specific — this one exists purely to carry the package doc
// comment on every GOOS.
//
// Testing: on darwin/linux the tests open a real pseudo-terminal (via
// /dev/ptmx and the pty ioctls, no dependencies) and check the termios
// flags and the resulting line-discipline behaviour; they skip where a pty
// can't be opened. The windows Console API path has no such harness and is
// verified by hand.
package term
