package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// parseHex reads "#RGB", "#RRGGBB" or either without the "#".
func parseHex(s string) (ansi.RGB, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return ansi.RGB{}, false
	}
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return ansi.RGB{}, false
	}
	return ansi.RGB{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n)}, true
}

func hex(c ansi.RGB) string { return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B) }

var (
	white = ansi.RGB{R: 255, G: 255, B: 255}
	black = ansi.RGB{}
)

// grade is the WCAG 2 verdict for text of colour fg on bg: AAA at 7, AA at
// 4.5, "AA large" at 3 (large or bold text only), otherwise "fail".
func grade(ratio float64) string {
	switch {
	case ratio >= 7:
		return "AAA"
	case ratio >= 4.5:
		return "AA"
	case ratio >= 3:
		return "AA large"
	}
	return "fail"
}

func contrast(fg, bg ansi.RGB) float64 { return theme.Contrast(fg, bg) }

// swatch is one entry of the derived palette.
type swatch struct {
	name string
	c    ansi.RGB
}

// palette derives six colours from c: two darker, two lighter, the colour
// itself and its complement. The order is the one they are shown in.
func palette(c ansi.RGB) []swatch {
	return []swatch{
		{"dark-2", c.Darken(0.4)}, {"dark-1", c.Darken(0.2)}, {"base", c},
		{"light-1", c.Lighten(0.2)}, {"light-2", c.Lighten(0.4)}, {"complement", c.Complement()},
	}
}

// css is the palette as custom properties, one per line.
func css(ps []swatch) string {
	var b strings.Builder
	b.WriteString(":root {\n")
	for _, p := range ps {
		fmt.Fprintf(&b, "  --color-%s: %s;\n", p.name, strings.ToLower(hex(p.c)))
	}
	b.WriteString("}\n")
	return b.String()
}
