package widgets

import "strings"

// Spacer returns a block of exactly height blank lines, each exactly width
// spaces wide. width <= 0 or height <= 0 returns "" rather than panicking or
// producing negative-sized output.
func Spacer(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	line := strings.Repeat(" ", width)
	lines := make([]string, height)
	for i := range lines {
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}
