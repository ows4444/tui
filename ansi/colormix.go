package ansi

// Mix returns c moved toward other by t: t = 0 is c, t = 1 is other, and t
// outside 0..1 is clamped to it. Each channel is interpolated linearly in sRGB
// and rounded to the nearest integer. Mixing a colour with the background it
// is drawn on, by an opacity, is how a translucent colour is made.
func (c RGB) Mix(other RGB, t float64) RGB {
	t = min(max(t, 0), 1)
	return RGB{R: mixByte(c.R, other.R, t), G: mixByte(c.G, other.G, t), B: mixByte(c.B, other.B, t)}
}

// Lighten returns c mixed toward white by t (0..1): 0 is c, 1 is white.
func (c RGB) Lighten(t float64) RGB { return c.Mix(RGB{R: 255, G: 255, B: 255}, t) }

// Darken returns c mixed toward black by t (0..1): 0 is c, 1 is black.
func (c RGB) Darken(t float64) RGB { return c.Mix(RGB{}, t) }

// Blend returns n colours spread evenly from the first stop to the last, with
// the stops themselves at even intervals between: the colours of a gradient
// laid over n cells. With no stops or n < 1 it returns nil, with one stop n
// copies of it, and with n = 1 the first stop.
func Blend(n int, stops ...RGB) []RGB {
	if n < 1 || len(stops) == 0 {
		return nil
	}
	out := make([]RGB, n)
	if len(stops) == 1 || n == 1 {
		for i := range out {
			out[i] = stops[0]
		}
		return out
	}
	segments := len(stops) - 1
	for i := range out {
		pos := float64(i) / float64(n-1) * float64(segments)
		lo := min(int(pos), segments-1)
		out[i] = stops[lo].Mix(stops[lo+1], pos-float64(lo))
	}
	return out
}

func mixByte(a, b uint8, t float64) uint8 {
	return uint8(float64(a) + (float64(b)-float64(a))*t + 0.5)
}
