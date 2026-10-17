package chart

import (
	"math/bits"

	"github.com/ows4444/tui/internal/braille"
	"github.com/ows4444/tui/theme"
)

// Marker selects the glyphs a Canvas draws with, which also sets how many dots
// fit in one terminal cell. Braille packs the most; the block markers are
// chunkier but show in fonts and terminals where braille is faint or missing.
type Marker int

// The markers. Braille is the zero value, so a Canvas or Plot that does not
// ask for one draws as it always did.
const (
	// Braille is 2 columns by 4 rows of dots per cell.
	Braille Marker = iota
	// Sextant is 2 by 3, drawn with the Symbols for Legacy Computing block
	// (Unicode 13), which not every font has.
	Sextant
	// Quadrant is 2 by 2, drawn with the quadrant block elements.
	Quadrant
	// HalfBlock is 1 by 2, drawn with the upper and lower half blocks.
	HalfBlock
	// Block is 1 by 1, a full block.
	Block
)

// Dots returns how many dot columns and rows one cell holds.
func (m Marker) Dots() (cols, rows int) {
	switch m {
	case Sextant:
		return 2, 3
	case Quadrant:
		return 2, 2
	case HalfBlock:
		return 1, 2
	case Block:
		return 1, 1
	}
	return 2, 4
}

// String names the marker.
func (m Marker) String() string {
	switch m {
	case Sextant:
		return "sextant"
	case Quadrant:
		return "quadrant"
	case HalfBlock:
		return "half-block"
	case Block:
		return "block"
	}
	return "braille"
}

// quadrants are the 16 quadrant glyphs by pattern, bit 0 the upper left, 1 the
// upper right, 2 the lower left and 3 the lower right.
var quadrants = [16]string{" ", "▘", "▝", "▀", "▖", "▌", "▞", "▛", "▗", "▚", "▐", "▜", "▄", "▙", "▟", "█"}

// halfBlocks are the four half-block glyphs by pattern, bit 0 the top and bit 1
// the bottom.
var halfBlocks = [4]string{" ", "▀", "▄", "█"}

// sextant returns the glyph for a 2x3 pattern, bit 0 the upper left, 1 the
// upper right, 2 the middle left, 3 the middle right, 4 the lower left and 5
// the lower right. The Unicode block lists the 60 patterns that are not blank,
// full or a half block in order; the half blocks and the full block are the
// older characters.
func sextant(n int) string {
	switch n {
	case 0:
		return " "
	case 21:
		return "▌"
	case 42:
		return "▐"
	case 63:
		return "█"
	}
	k := n - 1
	if n > 21 {
		k--
	}
	if n > 42 {
		k--
	}
	return string(rune(0x1FB00 + k)) // #nosec G115 -- k is 0 to 59, so the result is U+1FB00 to U+1FB3B
}

// cell returns the glyph for pattern, the bits of the dots set in one cell, or
// under an ASCII glyph set a shade chosen by how many of the cell's dots are set.
func (m Marker) cell(pattern rune, g theme.Glyphs) string {
	if m == Braille {
		return braille.DotCell(pattern, g)
	}
	cols, rows := m.Dots()
	if g.ASCII() {
		n := bits.OnesCount32(uint32(pattern)) // #nosec G115 -- pattern holds at most 6 bits
		if n == 0 {
			return " "
		}
		shades := []rune(g.Shades)
		return string(shades[min((n-1)*len(shades)/(cols*rows), len(shades)-1)])
	}
	switch m {
	case Sextant:
		return sextant(int(pattern))
	case Quadrant:
		return quadrants[pattern&15]
	case HalfBlock:
		return halfBlocks[pattern&3]
	}
	return "█"
}
