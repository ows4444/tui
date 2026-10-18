package avatar

import (
	"math"

	"github.com/ows4444/tui/ansi"
)

// A tint is where an expression pulls the body's colour: toward hue h, a
// share pull of the way to lightness l, with a chroma of at least c. A pose
// goes heat of the way to that colour, so the avatar keeps something of its
// own.
type tint struct{ h, l, pull, c float64 }

// The tints. Each is far enough from the others in hue or in pull that no
// two poses are told apart by tint alone.
var (
	tintHot   = &tint{h: 27, l: 0.58, pull: 0.6, c: 0.18}   // anger
	tintRose  = &tint{h: 358, l: 0.72, pull: 0.55, c: 0.16} // love
	tintBlush = &tint{h: 12, l: 0.84, pull: 0.4, c: 0.1}    // shyness
	tintBile  = &tint{h: 142, l: 0.66, pull: 0.6, c: 0.13}  // sickness
)

// oklchOf converts an sRGB colour to OKLCh.
func oklchOf(c ansi.RGB) oklch {
	lin := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	r, g, b := lin(c.R), lin(c.G), lin(c.B)
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	a := 1.9779984951*l - 2.428592205*m + 0.4505937099*s
	bb := 0.0259040371*l + 0.7827717662*m - 0.808675766*s
	return oklch{
		l: 0.2104542553*l + 0.793617785*m - 0.0040720468*s,
		c: math.Hypot(a, bb),
		h: math.Atan2(bb, a) * 180 / math.Pi,
	}
}

// mixRGB is the colour t of the way from a to b, mixed in OKLab so the path
// between two hues does not pass through grey.
func mixRGB(a, b ansi.RGB, t float64) ansi.RGB {
	x, y := oklchOf(a), oklchOf(b)
	ax, ay := x.c*math.Cos(rad(x.h)), x.c*math.Sin(rad(x.h))
	bx, by := y.c*math.Cos(rad(y.h)), y.c*math.Sin(rad(y.h))
	mx, my := ax+(bx-ax)*t, ay+(by-ay)*t
	return oklch{
		l: x.l + (y.l-x.l)*t,
		c: math.Hypot(mx, my),
		h: math.Atan2(my, mx) * 180 / math.Pi,
	}.rgb()
}

// target returns the body colour at the far end of tint t for a body colour
// head on ground ref: pulled to the tint, then held to bodyFloor against the
// ground.
func (t *tint) target(head ansi.RGB, ref oklch) ansi.RGB {
	base := oklchOf(head)
	return ensureContrast(oklch{l: base.l + (t.l-base.l)*t.pull, c: math.Max(base.c, t.c), h: t.h}, ref, bodyFloor).rgb()
}

// tinted returns p with its body moved heat of the way toward t, on ground g,
// and kept at bodyFloor against the ground at every step of the way. The eyes
// are not mixed: they are chosen for the body as it is, as paletteFor does, so
// they clear 4.5:1 at every step.
func (p palette) tinted(t *tint, heat float64, g ground) palette {
	ref := g.ref(p.bg)
	hue := oklchOf(p.eye).h
	head := ensureContrast(oklchOf(mixRGB(p.head, t.target(p.head, ref), heat)), ref, bodyFloor)
	p.head, p.eye = head.rgb(), eyeFor(head, hue).rgb()
	return p
}
