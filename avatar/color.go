package avatar

import (
	"math"

	"github.com/ows4444/tui/ansi"
)

// Colours are chosen in OKLCh, where equal steps look equal, and resolved to
// sRGB here. Every hue gets the same lightness and chroma for its tone, so no
// name draws a muddier avatar than another.

type oklch struct{ l, c, h float64 }

// linear converts to linear-light sRGB, which may be out of gamut.
func (o oklch) linear() [3]float64 {
	r := rad(o.h)
	a := o.c * math.Cos(r)
	b := o.c * math.Sin(r)
	l := o.l + 0.3963377774*a + 0.2158037573*b
	m := o.l - 0.1055613458*a - 0.0638541728*b
	s := o.l - 0.0894841775*a - 1.291485548*b
	l, m, s = l*l*l, m*m*m, s*s*s
	return [3]float64{
		4.0767416621*l - 3.3077115913*m + 0.2309699292*s,
		-1.2684380046*l + 2.6097574011*m - 0.3413193965*s,
		-0.0041960863*l - 0.7034186147*m + 1.707614701*s,
	}
}

func inGamut(rgb [3]float64) bool {
	for _, v := range rgb {
		if v < -1e-4 || v > 1+1e-4 {
			return false
		}
	}
	return true
}

// resolve is linear with the chroma reduced, by bisection, until the colour
// fits sRGB; lightness and hue are kept.
func (o oklch) resolve() [3]float64 {
	rgb := o.linear()
	if !inGamut(rgb) {
		lo, hi := 0.0, o.c
		for i := 0; i < 12; i++ {
			mid := (lo + hi) / 2
			if inGamut((oklch{o.l, mid, o.h}).linear()) {
				lo = mid
			} else {
				hi = mid
			}
		}
		rgb = (oklch{o.l, lo, o.h}).linear()
	}
	for i, v := range rgb {
		rgb[i] = math.Min(1, math.Max(0, v))
	}
	return rgb
}

func (o oklch) luminance() float64 {
	c := o.resolve()
	return 0.2126*c[0] + 0.7152*c[1] + 0.0722*c[2]
}

// contrast is the WCAG contrast ratio of a and b.
func contrast(a, b oklch) float64 {
	x, y := a.luminance(), b.luminance()
	return (math.Max(x, y) + 0.05) / (math.Min(x, y) + 0.05)
}

// ensureContrast moves fg's lightness away from bg, in steps of 0.02, until
// the pair reaches min; if neither direction gets there it returns black or
// white, whichever contrasts more.
func ensureContrast(fg, bg oklch, min float64) oklch {
	if contrast(fg, bg) >= min {
		return fg
	}
	lean := -1.0
	if fg.l >= bg.l {
		lean = 1
	}
	for _, dir := range []float64{lean, -lean} {
		probe := fg
		for i := 0; i < 60; i++ {
			probe.l = math.Min(1, math.Max(0, probe.l+dir*0.02))
			if contrast(probe, bg) >= min {
				return probe
			}
			if probe.l == 0 || probe.l == 1 {
				break
			}
		}
	}
	black, white := oklch{0, 0, fg.h}, oklch{1, 0, fg.h}
	if contrast(black, bg) >= contrast(white, bg) {
		return black
	}
	return white
}

// rgb is the gamma-encoded 8-bit colour.
func (o oklch) rgb() ansi.RGB {
	c := o.resolve()
	var out [3]uint8
	for i, v := range c {
		s := 12.92 * v
		if v > 0.0031308 {
			s = 1.055*math.Pow(v, 1/2.4) - 0.055
		}
		out[i] = uint8(math.Floor(s*255 + 0.5)) // #nosec G115 -- s is in [0, 1]
	}
	return ansi.RGB{R: out[0], G: out[1], B: out[2]}
}

// tones is the swatch set, pale to ink: the upper edge of each band in
// [0, 1) with its lightness and chroma.
var tones = [...]struct{ upTo, l, c float64 }{
	{0.2, 0.86, 0.085},  // pastel
	{0.36, 0.9, 0.028},  // pale neutral
	{0.62, 0.73, 0.135}, // mid
	{0.8, 0.62, 0.165},  // deep
	{0.93, 0.87, 0.16},  // bright
	{1.0, 0.34, 0.035},  // ink
}

// darkSurface is the darkest page the body must stay visible on.
var darkSurface = oklch{0.145, 0, 0}

// palette is the three colours of an avatar.
type palette struct{ bg, head, eye ansi.RGB }

// paletteFor resolves the colours for a hue in degrees and a tone position in
// [0, 1). The eyes flip to light on a dark body, and two contrast floors are
// enforced: the body against the backdrop (1.25) and the eyes against the
// body (4.5).
func paletteFor(hue, tone float64) palette {
	t := tones[0]
	for _, b := range tones {
		if tone < b.upTo {
			t = b
			break
		}
	}
	bg := oklch{0.965, 0.01, hue}
	head := ensureContrast(oklch{t.l, t.c, hue}, darkSurface, 1.5)
	eye := oklch{0.97, 0.012, hue}
	if head.l >= 0.5 {
		eye = oklch{0.17, 0.02, hue}
	}
	head = ensureContrast(head, bg, 1.25)
	eye = ensureContrast(eye, head, 4.5)
	return palette{bg: bg.rgb(), head: head.rgb(), eye: eye.rgb()}
}

const hexDigits = "0123456789abcdef"

// hex formats c as #rrggbb.
func hex(c ansi.RGB) string {
	b := [7]byte{'#'}
	for i, v := range [3]uint8{c.R, c.G, c.B} {
		b[1+2*i], b[2+2*i] = hexDigits[v>>4], hexDigits[v&15]
	}
	return string(b[:])
}
