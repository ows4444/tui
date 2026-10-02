package term

// Console input-mode flags from the Win32 Console API (wincon.h). The
// arithmetic on them lives here, without a build tag, so it is unit-tested on
// every platform; only the syscalls that apply it are Windows-only.
const (
	enableProcessedInput       = 0x0001
	enableLineInput            = 0x0002
	enableEchoInput            = 0x0004
	enableWindowInput          = 0x0008
	enableQuickEditMode        = 0x0040
	enableExtendedFlags        = 0x0080
	enableVirtualTerminalInput = 0x0200
)

// rawInputMode returns the console input mode for raw operation given the
// original mode: no line buffering, echo or console Ctrl-C handling, virtual
// terminal input (keys arrive as the escape sequences input.Reader parses),
// and window-buffer-size events queued (so a resize is seen as an event, not
// only by polling). When mouse is true QuickEdit is also cleared and
// ENABLE_EXTENDED_FLAGS set, without which the console keeps QuickEdit on and
// swallows mouse clicks for text selection. With mouse false the QuickEdit and
// extended-flags bits are left exactly as they were.
func rawInputMode(orig uint32, mouse bool) uint32 {
	m := orig
	m &^= enableEchoInput | enableLineInput | enableProcessedInput
	m |= enableVirtualTerminalInput | enableWindowInput
	if mouse {
		m &^= enableQuickEditMode
		m |= enableExtendedFlags
	}
	return m
}

// makeRawOn reads the current mode of fd on c, sets rawInputMode, and returns
// the original mode, which restoring must reapply unchanged.
func makeRawOn(c Console, fd int, mouse bool) (orig uint32, err error) {
	orig, err = c.Mode(fd)
	if err != nil {
		return 0, err
	}
	if err := c.SetMode(fd, rawInputMode(orig, mouse)); err != nil {
		return 0, err
	}
	return orig, nil
}
