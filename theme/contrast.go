package theme

import (
	"fmt"
	"math"

	"github.com/ows4444/tui/ansi"
)

// xtermBasic is xterm's default palette for the 16 named ANSI colours. A real
// terminal shows whatever its own scheme says, so ratios for BasicColor and
// Color256 values are a reference point, not a guarantee.
var xtermBasic = [16][3]uint8{
	{0x00, 0x00, 0x00}, {0xCD, 0x00, 0x00}, {0x00, 0xCD, 0x00}, {0xCD, 0xCD, 0x00},
	{0x00, 0x00, 0xEE}, {0xCD, 0x00, 0xCD}, {0x00, 0xCD, 0xCD}, {0xE5, 0xE5, 0xE5},
	{0x7F, 0x7F, 0x7F}, {0xFF, 0x00, 0x00}, {0x00, 0xFF, 0x00}, {0xFF, 0xFF, 0x00},
	{0x5C, 0x5C, 0xFF}, {0xFF, 0x00, 0xFF}, {0x00, 0xFF, 0xFF}, {0xFF, 0xFF, 0xFF},
}

// Palette is the terminal's ANSI 0-15 colours. Start from NewPalette (xterm's
// defaults) and feed it the terminal's OSC 4 answers with Observe. A Palette
// is not safe for concurrent use.
type Palette struct {
	c        [16][3]uint8
	reported int // bitmask of slots the terminal answered
}

// NewPalette returns xterm's default palette, the assumption used when the
// terminal does not answer OSC 4.
func NewPalette() *Palette { return &Palette{c: xtermBasic} }

// Set records the terminal's colour for slot index (0-15); other slots are
// ignored.
func (p *Palette) Set(index, r, g, b uint8) {
	if index < 16 {
		p.c[index] = [3]uint8{r, g, b}
		p.reported |= 1 << index
	}
}

// Observe records msg if it is a tui.PaletteColorEvent (the answer to
// ansi.QueryPalette) and reports whether it was one. Call it from Update.
func (p *Palette) Observe(msg any) bool {
	m, ok := msg.(paletteColor)
	if !ok {
		return false
	}
	i, r, g, b := m.PaletteColor()
	p.Set(i, r, g, b)
	return true
}

// Reported reports whether the terminal answered for slot index.
func (p *Palette) Reported(index uint8) bool { return index < 16 && p.reported&(1<<index) != 0 }

// Contrast is the package Contrast measured with this palette's colours for
// named ANSI and the first 16 of the 256-colour values. A nil Palette means
// xterm's defaults.
func (p *Palette) Contrast(fg, bg ansi.Color) float64 {
	return contrastIn(p.table(), fg, bg)
}

// Check is Theme.Check with this palette.
func (p *Palette) Check(t Theme, min float64) []ContrastIssue {
	return t.check(p.table(), min)
}

func (p *Palette) table() *[16][3]uint8 {
	if p == nil {
		return &xtermBasic
	}
	return &p.c
}

type paletteColor interface {
	PaletteColor() (index, r, g, b uint8)
}

// toRGB resolves c to RGB; ok is false for nil or unknown colour types.
func toRGB(c ansi.Color) (r, g, b uint8, ok bool) { return toRGBIn(&xtermBasic, c) }

func toRGBIn(pal *[16][3]uint8, c ansi.Color) (r, g, b uint8, ok bool) {
	switch v := c.(type) {
	case ansi.RGB:
		return v.R, v.G, v.B, true
	case ansi.BasicColor:
		if int(v) < len(xtermBasic) {
			p := pal[v]
			return p[0], p[1], p[2], true
		}
	case ansi.Color256:
		n := int(v)
		switch {
		case n < 16:
			p := pal[n]
			return p[0], p[1], p[2], true
		case n < 232:
			n -= 16
			lv := func(i int) uint8 {
				if i == 0 {
					return 0
				}
				return uint8(55 + 40*i) // #nosec G115 -- i is 1..5, so at most 255
			}
			return lv(n / 36), lv(n / 6 % 6), lv(n % 6), true
		default:
			l := uint8(8 + 10*(n-232)) // #nosec G115 -- n is 232..255, so at most 238
			return l, l, l, true
		}
	}
	return 0, 0, 0, false
}

func luminance(r, g, b uint8) float64 {
	lin := func(c uint8) float64 {
		cs := float64(c) / 255
		if cs <= 0.03928 {
			return cs / 12.92
		}
		return math.Pow((cs+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// Contrast returns the WCAG 2.x contrast ratio (1 to 21) between fg and bg.
// Named ANSI and 256-colour values are measured against xterm's default
// palette. It returns 0 if either colour is nil or of an unknown type.
func Contrast(fg, bg ansi.Color) float64 { return contrastIn(&xtermBasic, fg, bg) }

func contrastIn(pal *[16][3]uint8, fg, bg ansi.Color) float64 {
	fr, fgc, fb, ok1 := toRGBIn(pal, fg)
	br, bgc, bb, ok2 := toRGBIn(pal, bg)
	if !ok1 || !ok2 {
		return 0
	}
	l1, l2 := luminance(fr, fgc, fb), luminance(br, bgc, bb)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// ContrastIssue is one colour role that falls short of the requested ratio.
type ContrastIssue struct {
	Role  string // field name, e.g. "Muted"
	Fg    ansi.Color
	Bg    ansi.Color
	Ratio float64
}

// String formats the issue as "Role: contrast R.RR".
func (i ContrastIssue) String() string {
	return fmt.Sprintf("%s: contrast %.2f", i.Role, i.Ratio)
}

// Check reports every text and accent role (Text, Primary, Secondary,
// Success, Warning, Error, Info, Muted, Focus) whose contrast against the
// theme's Background is below min, e.g. 4.5 for WCAG AA. A theme without a
// Background falls back to TextInverse. Unset (nil) roles are skipped. Use it
// in an app's CI to vet a custom theme.
func (t Theme) Check(min float64) []ContrastIssue { return t.check(&xtermBasic, min) }

// CheckOn is Check against bg instead of the theme's Background, for vetting
// a theme on Surface, Overlay or a terminal background you probed.
func (t Theme) CheckOn(bg ansi.Color, min float64) []ContrastIssue {
	return t.checkOn(&xtermBasic, bg, min)
}

// background is Background, or TextInverse for a theme that predates it.
func (t Theme) background() ansi.Color {
	if t.Background != nil {
		return t.Background
	}
	return t.TextInverse
}

func (t Theme) check(pal *[16][3]uint8, min float64) []ContrastIssue {
	return t.checkOn(pal, t.background(), min)
}

func (t Theme) checkOn(pal *[16][3]uint8, bg ansi.Color, min float64) []ContrastIssue {
	if bg == nil {
		return nil
	}
	roles := []struct {
		name string
		c    ansi.Color
	}{
		{"Text", t.Text}, {"Primary", t.Primary}, {"Secondary", t.Secondary},
		{"Success", t.Success}, {"Warning", t.Warning}, {"Error", t.Error},
		{"Info", t.Info}, {"Muted", t.Muted}, {"Focus", t.Focus},
	}
	var out []ContrastIssue
	for _, r := range roles {
		if r.c == nil {
			continue
		}
		if ratio := contrastIn(pal, r.c, bg); ratio < min {
			out = append(out, ContrastIssue{Role: r.name, Fg: r.c, Bg: bg, Ratio: ratio})
		}
	}
	return out
}
