package ansi

import (
	"math"
	"strconv"
	"strings"
)

// FilterColors rewrites every colour in the SGR sequences of s through f and
// returns the result. A foreground, background or underline colour, whether
// written as one of the 16 basic codes, a 256-colour index or an RGB triple, is
// turned into an RGB, passed to f, and written back as a truecolor code;
// downgrade the result with DowngradeString for a smaller profile. The basic
// and 256-colour indexes are read as the xterm palette, since the real colours
// are the terminal's to choose. Default colours (SGR 39, 49, 59), attributes,
// text and every other sequence pass through unchanged.
//
// It is what Program applies under WithColorFilter. Grayscale, Protanopia,
// Deuteranopia, Tritanopia and Dim are ready filters.
func FilterColors(s string, f func(RGB) RGB) string {
	if f == nil || !strings.Contains(s, "\x1b[") {
		return s
	}
	b := make([]byte, 0, len(s)+len(s)/8)
	for i := 0; i < len(s); {
		if s[i] != 0x1b || i+1 >= len(s) || s[i+1] != '[' {
			b = append(b, s[i])
			i++
			continue
		}
		j := i + 2
		for j < len(s) && (s[j] < 0x40 || s[j] > 0x7E) {
			j++
		}
		if j >= len(s) { // unterminated: leave as is
			b = append(b, s[i:]...)
			break
		}
		if s[j] == 'm' {
			b = filterSGR(b, s[i+2:j], f)
		} else {
			b = append(b, s[i:j+1]...)
		}
		i = j + 1
	}
	return string(b)
}

// filterSGR appends to dst the escape sequence for the SGR parameters params
// with each colour passed through f.
func filterSGR(dst []byte, params string, f func(RGB) RGB) []byte {
	if params == "" {
		return append(dst, "\x1b[m"...)
	}
	w := sgrWriter{b: dst}
	rgb := func(base string, c RGB) {
		c = f(c)
		w.param(base)
		w.param("2")
		w.int(int(c.R))
		w.int(int(c.G))
		w.int(int(c.B))
	}
	toRGB := func(c sgrColor) RGB {
		if c.isRGB {
			return c.rgb
		}
		return c.idx.RGB()
	}
	toks := sgrParams{s: params}
	for t, ok := toks.next(); ok; t, ok = toks.next() {
		if k := strings.IndexByte(t, ':'); k > 0 {
			if base := t[:k]; base == "38" || base == "48" || base == "58" {
				if c, ok := parseColonColor(t[k+1:]); ok {
					rgb(base, toRGB(c))
					continue
				}
			}
			w.param(t)
			continue
		}
		n, err := strconv.Atoi(t)
		if err != nil {
			w.param(t)
			continue
		}
		switch {
		case n == 38 || n == 48 || n == 58:
			pos := toks.pos
			if c, ok := parseSemiColor(&toks); ok {
				rgb(t, toRGB(c))
			} else {
				// A spec we cannot read: keep the code and let its arguments
				// follow as ordinary parameters.
				toks.pos = pos
				w.param(t)
			}
		case n >= 30 && n <= 37:
			rgb("38", BasicColor(n-30).RGB())
		case n >= 90 && n <= 97:
			rgb("38", BasicColor(n-90+8).RGB())
		case n >= 40 && n <= 47:
			rgb("48", BasicColor(n-40).RGB())
		case n >= 100 && n <= 107:
			rgb("48", BasicColor(n-100+8).RGB())
		default:
			w.param(t)
		}
	}
	if w.n > 0 {
		w.b = append(w.b, 'm')
	}
	return w.b
}

// srgbToLinear and linearToSRGB convert one channel between the gamma-encoded
// 0..255 value and light intensity 0..1.
func srgbToLinear(v uint8) float64 {
	c := float64(v) / 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func linearToSRGB(l float64) uint8 {
	l = min(max(l, 0), 1)
	if l <= 0.0031308 {
		return uint8(math.Round(l * 12.92 * 255))
	}
	return uint8(math.Round((1.055*math.Pow(l, 1/2.4) - 0.055) * 255))
}

// Grayscale returns c as the gray of the same luminance (Rec. 709 weights on
// linear light).
func Grayscale(c RGB) RGB {
	y := 0.2126*srgbToLinear(c.R) + 0.7152*srgbToLinear(c.G) + 0.0722*srgbToLinear(c.B)
	g := linearToSRGB(y)
	return RGB{R: g, G: g, B: g}
}

// matrix3 is a 3x3 matrix applied to a colour in linear light.
type matrix3 [3][3]float64

func (m matrix3) apply(c RGB) RGB {
	r, g, b := srgbToLinear(c.R), srgbToLinear(c.G), srgbToLinear(c.B)
	return RGB{
		R: linearToSRGB(m[0][0]*r + m[0][1]*g + m[0][2]*b),
		G: linearToSRGB(m[1][0]*r + m[1][1]*g + m[1][2]*b),
		B: linearToSRGB(m[2][0]*r + m[2][1]*g + m[2][2]*b),
	}
}

// The matrices below simulate full dichromacy (Machado, Oliveira and Fernandes,
// 2009, severity 1.0). Every row sums to 1, so a gray stays the same gray.
var (
	protanopia   = matrix3{{0.152286, 1.052583, -0.204868}, {0.114503, 0.786281, 0.099216}, {-0.003882, -0.048116, 1.051998}}
	deuteranopia = matrix3{{0.367322, 0.860646, -0.227968}, {0.280085, 0.672501, 0.047413}, {-0.011820, 0.042940, 0.968881}}
	tritanopia   = matrix3{{1.255528, -0.076749, -0.178779}, {-0.078411, 0.930809, 0.147602}, {0.004733, 0.691367, 0.303900}}
)

// Protanopia returns c as a person with no working red cones would see it. It
// is an approximation for checking a design, not a medical model.
func Protanopia(c RGB) RGB { return protanopia.apply(c) }

// Deuteranopia returns c as a person with no working green cones would see it
// (the most common kind of colour blindness). An approximation, like Protanopia.
func Deuteranopia(c RGB) RGB { return deuteranopia.apply(c) }

// Tritanopia returns c as a person with no working blue cones would see it. An
// approximation, like Protanopia.
func Tritanopia(c RGB) RGB { return tritanopia.apply(c) }

// Dim returns a filter that darkens a colour toward black: Dim(1) leaves it,
// Dim(0.5) halves its brightness and Dim(0) is black. A factor outside 0..1 is
// clamped to it.
func Dim(factor float64) func(RGB) RGB {
	factor = min(max(factor, 0), 1)
	return func(c RGB) RGB { return c.Darken(1 - factor) }
}
