package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/boxdraw"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Tooltip renders text in a small bordered box styled via t.Info (so a
// tooltip visibly differs from the neutral Box/Panel border color), the
// same width convention as Box/Alert: width == 0 sizes the box to text,
// a positive width wraps text so every rendered line's ansi.Width stays
// within it.
func Tooltip(text string, t theme.Theme, width int) string {
	cw, auto := boxContentWidth(width)

	body := text
	if !auto {
		body = ansi.WrapStyled(text, cw)
	}

	return boxdraw.Draw(layout.NewBox().BorderColor(t.Info), t.Border, 1, cw, body)
}

// tooltipSize returns the rendered width (the widest line) and height
// (line count) of a Tooltip's output, needed to decide whether it fits
// below/right of an anchor before compositing it with layout.Overlay.
func tooltipSize(rendered string) (w, h int) {
	lines := strings.Split(rendered, "\n")
	h = len(lines)
	for _, line := range lines {
		if lw := ansi.Width(line); lw > w {
			w = lw
		}
	}
	return w, h
}

// baseExtent returns base's rendered width (the widest line) and height
// (line count), used to keep a TooltipOverlay tooltip within base's
// bounds.
func baseExtent(base string) (w, h int) {
	lines := strings.Split(base, "\n")
	h = len(lines)
	for _, line := range lines {
		if lw := ansi.Width(line); lw > w {
			w = lw
		}
	}
	return w, h
}

// TooltipOverlay composites a Tooltip showing text onto base near the
// anchor point (anchorX, anchorY), via layout.Overlay.
//
// Default placement is below and right-aligned to the anchor: the
// tooltip's top-left corner starts at (anchorX, anchorY+1), so the
// tooltip sits directly under the anchor and grows to the right of it
// (its left edge, not its right edge, is what lines up with anchorX).
//
// If that default placement would push the tooltip's right edge
// (anchorX + tooltip width) past base's rendered width, the tooltip is
// shifted left just enough to stay within base's width instead of being
// clipped by Overlay.
//
// If placing the tooltip below the anchor would push it past base's
// last line (anchorY + 1 + tooltip height > base's line count), the
// tooltip is placed above the anchor instead, its bottom edge ending at
// the row just before anchorY.
func TooltipOverlay(base string, text string, anchorX, anchorY int, t theme.Theme) string {
	tooltip := Tooltip(text, t, 0)
	tw, th := tooltipSize(tooltip)
	bw, bh := baseExtent(base)

	x := anchorX
	if x+tw > bw {
		x = bw - tw
	}
	if x < 0 {
		x = 0
	}

	y := anchorY + 1
	if y+th > bh {
		y = anchorY - th
	}
	if y < 0 {
		y = 0
	}

	return layout.Overlay(base, tooltip, x, y)
}
