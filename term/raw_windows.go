//go:build windows

package term

// State holds the console's original mode so it can be restored on exit.
type State struct {
	mode uint32
}

// IsTerminal reports whether fd is connected to a console.
func IsTerminal(fd int) bool {
	_, err := getConsoleMode(fd)
	return err == nil
}

// MakeRaw puts the console connected to fd into raw mode: no line
// buffering, no echo, no Ctrl-C processing by the console itself, and
// virtual-terminal input enabled so arrow keys etc. arrive as the same
// ANSI escape sequences input.Reader already parses on darwin/linux,
// instead of as separate Windows INPUT_RECORD events. Window-buffer-size
// events are also enabled, for the input reader to report resizes.
func MakeRaw(fd int) (*State, error) { return MakeRawMouse(fd, false) }

// MakeRawMouse is MakeRaw that, when mouse is true, also disables the
// console's QuickEdit mode, which would otherwise capture mouse clicks for
// text selection before they reach the program. The returned State holds the
// original mode, so Restore brings QuickEdit back too. With mouse false the
// QuickEdit setting is not touched.
func MakeRawMouse(fd int, mouse bool) (*State, error) {
	orig, err := makeRawOn(winConsole{}, fd, mouse)
	if err != nil {
		return nil, err
	}
	return &State{mode: orig}, nil
}

// Restore reapplies a State captured by MakeRaw.
func Restore(fd int, state *State) error {
	return setConsoleMode(fd, state.mode)
}

// winConsole is the real Console, backed by kernel32.
type winConsole struct{}

func (winConsole) Mode(fd int) (uint32, error) { return getConsoleMode(fd) }
func (winConsole) SetMode(fd int, mode uint32) error {
	return setConsoleMode(fd, mode)
}

// EnableOutputVT turns on virtual-terminal processing for the console
// connected to fd (see EnableOutputVTOn) and returns the function that
// restores the original mode. When fd is not a console (a pipe, a file) there
// is nothing to enable and the returned function does nothing.
func EnableOutputVT(fd int) (func() error, error) {
	if !IsTerminal(fd) {
		return func() error { return nil }, nil
	}
	return EnableOutputVTOn(winConsole{}, fd)
}
