package widgets

import (
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Gradient renders text with each character individually colored,
// linearly interpolating across the given colors stops. The theme
// parameter is accepted for signature consistency with the rest of the
// widgets package (every widget here takes a theme.Theme) but is unused:
// gradients are defined entirely by the caller-supplied color stops, not
// by theme colors.
//
// With len(colors) == 0 the text is returned unstyled. With a single
// color the text is rendered uniformly in that color (the degenerate,
// no-gradient case). With two or more stops, character i of n total maps
// to position p = i/(n-1) (or 0 when n == 1) along the gradient; p is
// then scaled by len(colors)-1 to find the two nearest stops and the
// fractional distance between them, and the color is linearly
// interpolated (per channel, rounded to the nearest integer) between
// those two stops. This stretches the interpolation proportionally
// across the whole string even when text is longer than colors, rather
// than repeating or clamping to the nearest stop.
//
// bold=true applies bold styling on top of each character's interpolated
// color. Empty text returns "".
func Gradient(text string, colors []ansi.RGB, bold bool, t theme.Theme) string {
	_ = t // theme unused: gradient stops are caller-supplied, not theme-derived

	if text == "" {
		return ""
	}
	if len(colors) == 0 {
		return text
	}

	runes := []rune(text)
	n := len(runes)

	var out []byte
	for i, r := range runes {
		c := gradientColorAt(i, n, colors)
		style := ansi.NewStyle().Foreground(c)
		if bold {
			style = style.Bold()
		}
		out = append(out, style.Render(string(r))...)
	}
	return string(out)
}

// gradientColorAt returns the interpolated color for character index i of
// n total characters, across the given color stops. colors must have at
// least one element.
func gradientColorAt(i, n int, colors []ansi.RGB) ansi.RGB {
	if len(colors) == 1 {
		return colors[0]
	}

	var p float64
	if n > 1 {
		p = float64(i) / float64(n-1)
	}

	// Scale p (0..1) into stop-space (0..len(colors)-1).
	scaled := p * float64(len(colors)-1)
	lo := int(scaled)
	if lo >= len(colors)-1 {
		return colors[len(colors)-1]
	}
	hi := lo + 1
	frac := scaled - float64(lo)

	return lerpRGB(colors[lo], colors[hi], frac)
}

// lerpRGB linearly interpolates between a and b at fraction t (0..1),
// rounding each channel to the nearest integer.
func lerpRGB(a, b ansi.RGB, t float64) ansi.RGB {
	return ansi.RGB{
		R: lerpByte(a.R, b.R, t),
		G: lerpByte(a.G, b.G, t),
		B: lerpByte(a.B, b.B, t),
	}
}

func lerpByte(a, b uint8, t float64) uint8 {
	v := float64(a) + (float64(b)-float64(a))*t
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return uint8(v + 0.5)
}
