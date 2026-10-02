//go:build windows

package term

import "unsafe"

type coord struct {
	X, Y int16
}

type smallRect struct {
	Left, Top, Right, Bottom int16
}

type consoleScreenBufferInfo struct {
	Size              coord
	CursorPosition    coord
	Attributes        uint16
	Window            smallRect
	MaximumWindowSize coord
}

// GetSize returns the console's visible window width and height in
// columns/rows (the window can be smaller than the scrollback buffer,
// which Size/MaximumWindowSize describe instead).
func GetSize(fd int) (width, height int, err error) {
	var info consoleScreenBufferInfo
	r, _, e := procGetConsoleScreenBufferInfo.Call(uintptr(fd), uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 0, 0, e
	}
	width = int(info.Window.Right-info.Window.Left) + 1
	height = int(info.Window.Bottom-info.Window.Top) + 1
	return width, height, nil
}
