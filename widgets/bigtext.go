package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// Font selects the glyph style BigText renders with.
type Font int

const (
	// FontBlock is a solid 5-row block glyph set covering uppercase
	// A-Z, digits 0-9, and space.
	FontBlock Font = iota
	// FontSimple is a 5-row font drawn only with ASCII characters
	// (+ - | / \ and similar).
	FontSimple
	// FontShade is a 5-row block font drawn with the light, medium and dark
	// shade characters for depth.
	FontShade
	// FontSlim is a 3-row font drawn with thin box-drawing strokes.
	FontSlim
)

// bigTextHeight is the row count of every glyph in bigTextGlyphs, and of
// the blank glyph substituted for characters outside the lookup table.
const bigTextHeight = 5

// bigTextGap is the number of blank columns drawn between adjacent
// glyphs.
const bigTextGap = 1

// bigTextGlyphs maps each supported character to its bigTextHeight-row,
// equal-width block glyph. This is an original, hand-drawn glyph set
// designed for this package — a simple blocky look built from █/light
// shade characters, not a transcription of any existing figlet font's
// bitmaps.
var bigTextGlyphs = map[rune][]string{
	' ': {
		"    ",
		"    ",
		"    ",
		"    ",
		"    ",
	},
	'A': {
		" ██ ",
		"█  █",
		"████",
		"█  █",
		"█  █",
	},
	'B': {
		"███ ",
		"█  █",
		"███ ",
		"█  █",
		"███ ",
	},
	'C': {
		" ███",
		"█   ",
		"█   ",
		"█   ",
		" ███",
	},
	'D': {
		"███ ",
		"█  █",
		"█  █",
		"█  █",
		"███ ",
	},
	'E': {
		"████",
		"█   ",
		"███ ",
		"█   ",
		"████",
	},
	'F': {
		"████",
		"█   ",
		"███ ",
		"█   ",
		"█   ",
	},
	'G': {
		" ███",
		"█   ",
		"█ ██",
		"█  █",
		" ███",
	},
	'H': {
		"█  █",
		"█  █",
		"████",
		"█  █",
		"█  █",
	},
	'I': {
		"███",
		" █ ",
		" █ ",
		" █ ",
		"███",
	},
	'J': {
		"  ██",
		"   █",
		"   █",
		"█  █",
		" ██ ",
	},
	'K': {
		"█  █",
		"█ █ ",
		"██  ",
		"█ █ ",
		"█  █",
	},
	'L': {
		"█   ",
		"█   ",
		"█   ",
		"█   ",
		"████",
	},
	'M': {
		"█   █",
		"██ ██",
		"█ █ █",
		"█   █",
		"█   █",
	},
	'N': {
		"█   █",
		"██  █",
		"█ █ █",
		"█  ██",
		"█   █",
	},
	'O': {
		" ██ ",
		"█  █",
		"█  █",
		"█  █",
		" ██ ",
	},
	'P': {
		"███ ",
		"█  █",
		"███ ",
		"█   ",
		"█   ",
	},
	'Q': {
		" ██ ",
		"█  █",
		"█  █",
		"█ █ ",
		" ███",
	},
	'R': {
		"███ ",
		"█  █",
		"███ ",
		"█ █ ",
		"█  █",
	},
	'S': {
		" ███",
		"█   ",
		" ██ ",
		"   █",
		"███ ",
	},
	'T': {
		"█████",
		"  █  ",
		"  █  ",
		"  █  ",
		"  █  ",
	},
	'U': {
		"█  █",
		"█  █",
		"█  █",
		"█  █",
		" ██ ",
	},
	'V': {
		"█   █",
		"█   █",
		" █ █ ",
		" █ █ ",
		"  █  ",
	},
	'W': {
		"█   █",
		"█   █",
		"█ █ █",
		"██ ██",
		"█   █",
	},
	'X': {
		"█   █",
		" █ █ ",
		"  █  ",
		" █ █ ",
		"█   █",
	},
	'Y': {
		"█   █",
		" █ █ ",
		"  █  ",
		"  █  ",
		"  █  ",
	},
	'Z': {
		"████",
		"   █",
		"  █ ",
		" █  ",
		"████",
	},
	'0': {
		" ██ ",
		"█  █",
		"█  █",
		"█  █",
		" ██ ",
	},
	'1': {
		" █ ",
		"██ ",
		" █ ",
		" █ ",
		"███",
	},
	'2': {
		" ██ ",
		"█  █",
		"  █ ",
		" █  ",
		"████",
	},
	'3': {
		"████",
		"   █",
		" ██ ",
		"   █",
		"████",
	},
	'4': {
		"█  █",
		"█  █",
		"████",
		"   █",
		"   █",
	},
	'5': {
		"████",
		"█   ",
		"███ ",
		"   █",
		"███ ",
	},
	'6': {
		" ███",
		"█   ",
		"███ ",
		"█  █",
		" ██ ",
	},
	'7': {
		"████",
		"   █",
		"  █ ",
		" █  ",
		" █  ",
	},
	'8': {
		" ██ ",
		"█  █",
		" ██ ",
		"█  █",
		" ██ ",
	},
	'9': {
		" ██ ",
		"█  █",
		" ███",
		"   █",
		" ██ ",
	},
}

// bigTextSlim holds the hand-drawn 3-row, 3-column thin font. Strokes use
// box-drawing characters; bigTextASCIIMap degrades them to ASCII.
var bigTextSlim = map[rune][]string{
	' ': {"   ", "   ", "   "},
	'A': {"╭─╮", "├─┤", "╵ ╵"},
	'B': {"┌─╮", "├─┤", "└─╯"},
	'C': {"╭─╴", "│  ", "╰─╴"},
	'D': {"┌─╮", "│ │", "└─╯"},
	'E': {"┌─╴", "├─ ", "└─╴"},
	'F': {"┌─╴", "├─ ", "╵  "},
	'G': {"╭─╮", "│ ─", "╰─╯"},
	'H': {"╷ ╷", "├─┤", "╵ ╵"},
	'I': {"╶┬╴", " │ ", "╶┴╴"},
	'J': {" ╶┐", "  │", "╰─╯"},
	'K': {"│ ╱", "├─ ", "│ ╲"},
	'L': {"╷  ", "│  ", "└─╴"},
	'M': {"╭┬╮", "│││", "╵ ╵"},
	'N': {"┌╮╷", "│╰┤", "╵ ╵"},
	'O': {"╭─╮", "│ │", "╰─╯"},
	'P': {"┌─╮", "├─╯", "╵  "},
	'Q': {"╭─╮", "│ │", "╰─╲"},
	'R': {"┌─╮", "├┬╯", "╵ ╰"},
	'S': {"╭─╴", "╰─╮", "╶─╯"},
	'T': {"╶┬╴", " │ ", " ╵ "},
	'U': {"╷ ╷", "│ │", "╰─╯"},
	'V': {"│ │", "╲ ╱", " ╵ "},
	'W': {"╷ ╷", "│╷│", "╰┴╯"},
	'X': {"╲ ╱", " ╳ ", "╱ ╲"},
	'Y': {"╲ ╱", " │ ", " ╵ "},
	'Z': {"╶─┐", " ╱ ", "└─╴"},
	'0': {"╭─╮", "│╱│", "╰─╯"},
	'1': {"╶┐ ", " │ ", "╶┴╴"},
	'2': {"╶─╮", "╭─╯", "╰─╴"},
	'3': {"╶─╮", " ─┤", "╶─╯"},
	'4': {"╷ ╷", "╰─┤", "  ╵"},
	'5': {"┌─╴", "╰─╮", "╶─╯"},
	'6': {"╭─╴", "├─╮", "╰─╯"},
	'7': {"╶─┐", " ╱ ", "╵  "},
	'8': {"╭─╮", "├─┤", "╰─╯"},
	'9': {"╭─╮", "╰─┤", "╶─╯"},
}

// bigTextASCIIMap degrades the non-ASCII runes of FontShade and FontSlim to
// single-column ASCII stand-ins for ASCII themes. The full block is not
// listed: it follows the theme's BarFull instead.
var bigTextASCIIMap = map[rune]rune{
	'░': '.', '▒': '=', '▓': '#',
	'─': '-', '│': '|', '╴': '-', '╶': '-', '╷': '|', '╵': '|',
	'┌': '+', '┐': '+', '└': '+', '┘': '+', '╭': '+', '╮': '+', '╰': '+', '╯': '+',
	'├': '+', '┤': '+', '┬': '+', '┴': '+', '┼': '+',
	'╱': '/', '╲': '\\', '╳': 'X',
}

// bigTextFonts holds the glyph table of each font; Simple and Shade are
// derived from the Block bitmaps so every font shares one character set.
var bigTextFonts = map[Font]map[rune][]string{
	FontBlock:  bigTextGlyphs,
	FontSimple: deriveGlyphs(simpleCell),
	FontShade:  deriveGlyphs(shadeCell),
	FontSlim:   bigTextSlim,
}

// deriveGlyphs builds a font from the Block bitmaps by replacing every cell
// with cell(self, at), where self reports whether the cell is part of a
// stroke and at(dy, dx) whether the cell at that offset is (false off the
// glyph).
func deriveGlyphs(cell func(self bool, at func(dy, dx int) bool) rune) map[rune][]string {
	out := make(map[rune][]string, len(bigTextGlyphs))
	for ch, g := range bigTextGlyphs {
		grid := make([][]rune, len(g))
		for y, row := range g {
			grid[y] = []rune(row)
		}
		rows := make([]string, len(g))
		for y := range grid {
			cells := make([]rune, len(grid[y]))
			for x := range grid[y] {
				at := func(dy, dx int) bool {
					yy, xx := y+dy, x+dx
					return yy >= 0 && yy < len(grid) && xx >= 0 && xx < len(grid[yy]) && grid[yy][xx] == '█'
				}
				cells[x] = cell(grid[y][x] == '█', at)
			}
			rows[y] = string(cells)
		}
		out[ch] = rows
	}
	return out
}

// simpleCell draws a stroke cell with + - | / \ V ^ X, chosen from its
// neighbours; blank cells stay blank.
func simpleCell(self bool, at func(dy, dx int) bool) rune {
	if !self {
		return ' '
	}
	h, v := at(0, -1) || at(0, 1), at(-1, 0) || at(1, 0)
	switch {
	case h && v:
		return '+'
	case h:
		return '-'
	case v:
		return '|'
	}
	ul, ur, dl, dr := at(-1, -1), at(-1, 1), at(1, -1), at(1, 1)
	switch {
	case ul && ur && dl && dr:
		return 'X'
	case ul && ur:
		return 'V'
	case dl && dr:
		return '^'
	case ul || dr:
		return '\\'
	case ur || dl:
		return '/'
	}
	return 'o'
}

// shadeCell draws strokes dark, their top edge medium, and the blank cells
// to the right of or below a stroke light, like a drop shadow.
func shadeCell(self bool, at func(dy, dx int) bool) rune {
	switch {
	case self && !at(-1, 0):
		return '▒'
	case self:
		return '▓'
	case at(0, -1) || at(-1, 0):
		return '░'
	}
	return ' '
}

// bigTextSize returns the row count of font's glyphs.
func bigTextSize(font Font) int {
	if font == FontSlim {
		return 3
	}
	return bigTextHeight
}

// glyphFor looks up r's glyph in font's table (an unknown font uses
// FontBlock's), falling back to a blank glyph the width of a space when r
// isn't in the table.
func glyphFor(font Font, r rune) []string {
	tab, ok := bigTextFonts[font]
	if !ok {
		font, tab = FontBlock, bigTextGlyphs
	}
	if g, ok := tab[r]; ok {
		return g
	}
	w := 1
	if sp, ok := tab[' ']; ok && len(sp) > 0 {
		w = len([]rune(sp[0]))
	}
	rows := make([]string, bigTextSize(font))
	for i := range rows {
		rows[i] = strings.Repeat(" ", w)
	}
	return rows
}

// bigTextRows lays out text in font, joined with a column gap and with the
// glyph runes already degraded for t's glyph set. It carries no styling.
// text must be non-empty.
func bigTextRows(text string, font Font, t theme.Theme) []string {
	runes := []rune(text)
	glyphs := make([][]string, len(runes))
	for i, r := range runes {
		glyphs[i] = glyphFor(font, r)
	}

	gap := strings.Repeat(" ", bigTextGap)
	rows := make([]string, len(glyphs[0]))
	for row := range rows {
		var b strings.Builder
		for i, g := range glyphs {
			if i > 0 {
				b.WriteString(gap)
			}
			b.WriteString(g[row])
		}
		rows[row] = b.String()
	}
	gs := t.GlyphSet()
	if gs.ASCII() {
		for i, r := range rows {
			rows[i] = strings.Map(func(c rune) rune {
				if a, ok := bigTextASCIIMap[c]; ok {
					return a
				}
				return c
			}, r)
		}
	}
	// The block font is drawn with full blocks; an ASCII theme swaps them for
	// its fill glyph, which takes the same single column.
	if full := gs.BarFull; full != "█" {
		for i, r := range rows {
			rows[i] = strings.ReplaceAll(r, "█", full)
		}
	}
	return rows
}

// BigText renders text as multi-row, block-letter ASCII art, one glyph
// per character joined horizontally with a small column gap, styled in
// the theme's Primary color.
//
// Each font has its own glyph table covering uppercase A-Z, digits 0-9 and
// space: FontBlock (5 rows, full blocks), FontSimple (5 rows, ASCII only),
// FontShade (5 rows, light/medium/dark shades) and FontSlim (3 rows, thin
// box-drawing strokes). Under an ASCII theme every font degrades to ASCII:
// FontBlock uses the theme's BarFull, FontShade and FontSlim map their
// characters to ASCII stand-ins.
//
// Characters outside the glyph table (lowercase letters, punctuation,
// etc.) render as a blank glyph of the same height rather than panicking.
// Empty text returns "" without panicking.
func BigText(text string, font Font, t theme.Theme) string {
	if text == "" {
		return ""
	}
	rows := bigTextRows(text, font, t)
	style := ansi.NewStyle().Foreground(t.Primary)
	styled := make([]string, len(rows))
	for i, r := range rows {
		styled[i] = style.Render(r)
	}
	return strings.Join(styled, "\n")
}

// BigTextGradient is BigText with a colour gradient across the columns: the
// first column is from, the last is to, and the columns between interpolate
// linearly in RGB. Only styling differs from BigText, so the stripped output
// and every row width are identical. A single-column text is drawn in from.
func BigTextGradient(text string, font Font, t theme.Theme, from, to ansi.RGB) string {
	if text == "" {
		return ""
	}
	rows := bigTextRows(text, font, t)
	lerp := func(a, b uint8, i, n int) uint8 {
		if n == 0 {
			return a
		}
		return uint8((int(a)*(n-i) + int(b)*i + n/2) / n) // #nosec G115 -- a weighted mean of two uint8 values, for 0 <= i <= n
	}
	out := make([]string, len(rows))
	for y, r := range rows {
		cells := []rune(r)
		n := len(cells) - 1
		var b strings.Builder
		for x, c := range cells {
			col := ansi.RGB{R: lerp(from.R, to.R, x, n), G: lerp(from.G, to.G, x, n), B: lerp(from.B, to.B, x, n)}
			b.WriteString(ansi.NewStyle().Foreground(col).Render(string(c)))
		}
		out[y] = b.String()
	}
	return strings.Join(out, "\n")
}
