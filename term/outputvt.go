package term

import "fmt"

// enableVirtualTerminalProcessing is ENABLE_VIRTUAL_TERMINAL_PROCESSING
// (wincon.h), the output-mode flag that makes a Windows console interpret
// escape sequences instead of printing them.
const enableVirtualTerminalProcessing uint32 = 0x0004

// Console is the part of the Windows console API output-VT setup needs. It
// is an interface so the decision logic runs, and is tested, on every
// platform; only the Win32 binding is Windows-only.
type Console interface {
	// Mode returns the console mode of the output handle fd
	// (GetConsoleMode).
	Mode(fd int) (uint32, error)
	// SetMode sets the console mode of fd (SetConsoleMode).
	SetMode(fd int, mode uint32) error
}

// EnableOutputVTOn makes sure the output console fd interprets virtual
// terminal sequences, setting ENABLE_VIRTUAL_TERMINAL_PROCESSING if its mode
// lacks it. It returns a function that puts the original mode back; the
// function is a no-op when the flag was already set, and is safe to call more
// than once. When the console cannot enable VT processing it returns an error
// naming it and leaves the mode as it found it.
func EnableOutputVTOn(c Console, fd int) (restore func() error, err error) {
	old, err := c.Mode(fd)
	if err != nil {
		return nil, fmt.Errorf("term: read console output mode for virtual terminal processing: %w", err)
	}
	if old&enableVirtualTerminalProcessing != 0 {
		return func() error { return nil }, nil
	}
	if err := c.SetMode(fd, old|enableVirtualTerminalProcessing); err != nil {
		return nil, fmt.Errorf("term: console does not support virtual terminal processing (ENABLE_VIRTUAL_TERMINAL_PROCESSING): %w", err)
	}
	return func() error { return c.SetMode(fd, old) }, nil
}
