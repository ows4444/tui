package layout

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// Overlay composites overlay on top of base at column x, row y (both
// 0-indexed, relative to base's own top-left corner), replacing whatever
// was there while leaving the rest of base untouched.
//
// This exists because Program.render treats View()'s return value as the
// entire screen with no notion of layered content — a widget that needs
// to draw on top of the rest of the view (a modal dialog, a toast
// notification) has to composite that itself before returning from View,
// and this is the shared primitive for doing that instead of solving it
// separately per widget.
//
// x and y are clamped to >= 0. Rows of overlay past base's last line are
// dropped. A base row shorter than x is padded with spaces before the
// overlay is spliced in. Base content beyond the overlay's right edge is
// preserved — Overlay never truncates a base line to the overlay's width,
// only replaces the columns the overlay actually covers. Keeping every
// resulting line the same width (e.g. for a rectangular bordered box) is
// the caller's responsibility: size the overlay to fit within base's
// existing width before calling this.
func Overlay(base, overlay string, x, y int) string {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}

	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	for i, oline := range overlayLines {
		row := y + i
		if row >= len(baseLines) {
			break
		}

		oWidth := ansi.Width(oline)
		left := ansi.Truncate(baseLines[row], x)
		if pad := x - ansi.Width(left); pad > 0 {
			left += strings.Repeat(" ", pad)
		}
		right := ansi.TrimLeftWidth(baseLines[row], x+oWidth)
		baseLines[row] = left + oline + right
	}

	return strings.Join(baseLines, "\n")
}
