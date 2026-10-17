//go:build !darwin && !linux

package pty

import (
	"errors"
	"os"
)

// Supported reports whether pseudo-terminals can be opened on this system.
const Supported = false

// ErrUnsupported is what Open and SetSize return where there are no pseudo-terminals.
var ErrUnsupported = errors.New("pty: not supported on this system")

// Open is not available here.
func Open() (master, slave *os.File, err error) { return nil, nil, ErrUnsupported }

// SetSize is not available here.
func SetSize(*os.File, uint16, uint16) error { return ErrUnsupported }
