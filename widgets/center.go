package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// contentSize returns the width (widest line) and height (line count) of a
// multi-line string, for measuring how much padding Center needs to add.
func contentSize(content string) (width, height int) {
	lines := strings.Split(content, "\n")
	height = len(lines)
	for _, line := range lines {
		if w := ansi.Width(line); w > width {
			width = w
		}
	}
	return width, height
}

// Center returns content padded to exactly width columns and height lines,
// with the content horizontally and vertically centered inside that box.
// Horizontal padding is split evenly between left and right, with any odd
// extra column going on the right; vertical padding is split evenly between
// top and bottom, with any odd extra line going on the bottom. If width or
// height is less than or equal to content's own size on that axis, that
// axis is left unpadded and unchanged rather than truncated.
func Center(content string, width, height int) string {
	cw, ch := contentSize(content)

	lines := strings.Split(content, "\n")
	if width > cw {
		remaining := width - cw
		left := remaining / 2
		right := remaining - left
		padded := make([]string, len(lines))
		for i, line := range lines {
			lw := ansi.Width(line)
			lineRight := right + (cw - lw)
			padded[i] = strings.Repeat(" ", left) + line + strings.Repeat(" ", lineRight)
		}
		lines = padded
	}

	if height > ch {
		remaining := height - ch
		top := remaining / 2
		bottom := remaining - top
		blankWidth := width
		if blankWidth < cw {
			blankWidth = cw
		}
		blank := strings.Repeat(" ", blankWidth)
		result := make([]string, 0, height)
		for i := 0; i < top; i++ {
			result = append(result, blank)
		}
		result = append(result, lines...)
		for i := 0; i < bottom; i++ {
			result = append(result, blank)
		}
		lines = result
	}

	return strings.Join(lines, "\n")
}
