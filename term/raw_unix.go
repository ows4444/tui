//go:build linux || darwin || dragonfly || freebsd || netbsd || openbsd

package term

import "syscall"

// State holds the terminal's original termios settings so they can be
// restored on exit.
type State struct {
	termios syscall.Termios
}

// IsTerminal reports whether fd is connected to a terminal.
func IsTerminal(fd int) bool {
	var t syscall.Termios
	return ioctlGetTermios(fd, &t) == nil
}

// MakeRaw puts the terminal connected to fd into raw mode: no line
// buffering, no echo, no signal generation from Ctrl-C/Ctrl-Z, 8-bit clean.
// It returns the previous state so the caller can Restore it.
func MakeRaw(fd int) (*State, error) {
	var oldState syscall.Termios
	if err := ioctlGetTermios(fd, &oldState); err != nil {
		return nil, err
	}

	newState := oldState
	newState.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP |
		syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	newState.Oflag &^= syscall.OPOST
	newState.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	newState.Cflag &^= syscall.CSIZE | syscall.PARENB
	newState.Cflag |= syscall.CS8

	// Return each byte as soon as it arrives; no line discipline timeout.
	newState.Cc[syscall.VMIN] = 1
	newState.Cc[syscall.VTIME] = 0

	if err := ioctlSetTermios(fd, &newState); err != nil {
		return nil, err
	}
	return &State{termios: oldState}, nil
}

// MakeRawMouse is MakeRaw. The mouse flag only matters on Windows, where it
// disables QuickEdit; a unix terminal has nothing to switch off.
func MakeRawMouse(fd int, _ bool) (*State, error) { return MakeRaw(fd) }

// Restore reapplies a State captured by MakeRaw.
func Restore(fd int, state *State) error {
	return ioctlSetTermios(fd, &state.termios)
}

// EnableOutputVT is a no-op on Unix terminals, which interpret escape
// sequences natively; it exists so callers need no build tags.
func EnableOutputVT(fd int) (func() error, error) {
	return func() error { return nil }, nil
}
