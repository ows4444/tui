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

// tintFloor is the eye-to-body contrast a tinted palette keeps, at the tint
// and at every step on the way to it.
const tintFloor = 4.55

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

// target returns the body and eye colours at the far end of tint t for a
// palette: the body pulled to the tint, and the eyes moved in lightness
// until they clear tintFloor against it, and against every step between the
// palette and the target.
func (t *tint) target(head, eye ansi.RGB) (ansi.RGB, ansi.RGB) {
	base, baseEye := oklchOf(head), oklchOf(eye)
	hot := ensureContrast(oklch{l: base.l + (t.l-base.l)*t.pull, c: math.Max(base.c, t.c), h: t.h}, darkSurface, 1.5)
	hotEye := ensureContrast(baseEye, hot, tintFloor)
	dir := -1.0
	if hotEye.l >= hot.l {
		dir = 1
	}
	hotHead := hot.rgb()
	for pass := 0; pass < 40; pass++ {
		e := hotEye.rgb()
		worst := math.Inf(1)
		for i := 0; i <= 10; i++ {
			at := float64(i) / 10
			worst = math.Min(worst, contrast(oklchOf(mixRGB(eye, e, at)), oklchOf(mixRGB(head, hotHead, at))))
		}
		if worst >= tintFloor {
			return hotHead, e
		}
		l := math.Min(1, math.Max(0, hotEye.l+dir*0.02))
		if l == hotEye.l {
			return hotHead, e
		}
		hotEye.l = l
	}
	return hotHead, hotEye.rgb()
}

// tinted returns p with its body and eyes moved heat of the way toward t.
func (p palette) tinted(t *tint, heat float64) palette {
	head, eye := t.target(p.head, p.eye)
	p.head, p.eye = mixRGB(p.head, head, heat), mixRGB(p.eye, eye, heat)
	return p
}
