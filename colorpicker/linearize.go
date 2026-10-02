package colorpicker

import (
	"fmt"
	"strconv"

	"github.com/ows4444/tui/ansi"
)

var basicNames = [16]string{
	"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white",
	"bright black", "bright red", "bright green", "bright yellow",
	"bright blue", "bright magenta", "bright cyan", "bright white",
}

// describe names a colour in words: a name for the 16 standard colours,
// "colour N" for a 256-palette index, and a hex code for RGB.
func describe(c ansi.Color) string {
	switch v := c.(type) {
	case ansi.BasicColor:
		if int(v) < len(basicNames) {
			return basicNames[v]
		}
	case ansi.Color256:
		return "colour " + strconv.Itoa(int(v))
	case ansi.RGB:
		return fmt.Sprintf("#%02x%02x%02x", v.R, v.G, v.B)
	}
	return "custom colour"
}

// Linearize renders the picker as plain text for accessible output (see
// tui.Linearizer): one line per palette swatch with its colour in words, its
// position and ", selected" on the cursor swatch (none while the hex field
// has focus), then the hex input's own line.
func (m Model) Linearize() string {
	var out []string
	for i, c := range m.Palette {
		line := describe(c) + ", swatch " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(m.Palette))
		if i == m.cursor && !m.hexFocused {
			line += ", selected"
		}
		out = append(out, line)
	}
	if len(m.Palette) == 0 {
		out = append(out, "No colours")
	}
	out = append(out, "Hex: "+m.HexInput.Linearize())
	return joinLines(out)
}

func joinLines(lines []string) string {
	s := ""
	for i, l := range lines {
		if i > 0 {
			s += "\n"
		}
		s += l
	}
	return s
}
