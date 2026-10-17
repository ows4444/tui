package ansi

import "math"

// HSL returns c as hue in degrees [0, 360), saturation and lightness in
// [0, 1]. A grey has hue 0 and saturation 0.
func (c RGB) HSL() (h, s, l float64) {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	hi, lo := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l = (hi + lo) / 2
	d := hi - lo
	if d == 0 {
		return 0, 0, l
	}
	s = d / (1 - math.Abs(2*l-1))
	switch hi {
	case r:
		h = math.Mod((g-b)/d, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h, s, l
}

// FromHSL is the RGB with hue h in degrees (any value; it wraps), saturation
// s and lightness l in [0, 1] (clamped).
func FromHSL(h, s, l float64) RGB {
	s, l = min(max(s, 0), 1), min(max(l, 0), 1)
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return RGB{R: unit(r + m), G: unit(g + m), B: unit(b + m)}
}

func unit(v float64) uint8 { return uint8(math.Round(min(max(v, 0), 1) * 255)) } // #nosec G115 -- clamped to [0, 255]

// Complement is the colour opposite c on the hue wheel: the hue turned half a
// circle, saturation and lightness kept.
func (c RGB) Complement() RGB {
	h, s, l := c.HSL()
	return FromHSL(h+180, s, l)
}

// Luminance is c's relative luminance as WCAG 2 defines it, 0 for black to 1
// for white.
func (c RGB) Luminance() float64 {
	lin := func(v uint8) float64 {
		f := float64(v) / 255
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// Readable is black or white, whichever reads better as text drawn on c: the
// one with the higher WCAG contrast ratio.
func (c RGB) Readable() RGB {
	l := c.Luminance()
	// Contrast with white is 1.05/(l+0.05), with black (l+0.05)/0.05.
	if 1.05/(l+0.05) >= (l+0.05)/0.05 {
		return RGB{R: 255, G: 255, B: 255}
	}
	return RGB{}
}
