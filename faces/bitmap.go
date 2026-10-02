package faces

import "strings"

// A sprite is drawn on a grid of dots and printed as Braille: each terminal
// cell holds a 2-wide, 4-tall block of dots (U+2800..U+28FF), so a 6x3-cell
// head is 12x12 dots.

// art is a small picture; every '#' is a dot.
type art []string

func (a art) width() int {
	w := 0
	for _, r := range a {
		w = max(w, len(r))
	}
	return w
}

// mirror flips a horizontally.
func (a art) mirror() art {
	w := a.width()
	out := make(art, len(a))
	for i, row := range a {
		row += strings.Repeat(".", w-len(row))
		b := []byte(row)
		for l, r := 0, len(b)-1; l < r; l, r = l+1, r-1 {
			b[l], b[r] = b[r], b[l]
		}
		out[i] = string(b)
	}
	return out
}

type bitmap struct {
	w, h int
	px   []bool
}

func newBitmap(w, h int) *bitmap { return &bitmap{w: w, h: h, px: make([]bool, w*h)} }

func (b *bitmap) in(x, y int) bool { return x >= 0 && y >= 0 && x < b.w && y < b.h }

func (b *bitmap) get(x, y int) bool { return b.in(x, y) && b.px[y*b.w+x] }

func (b *bitmap) set(x, y int, v bool) {
	if b.in(x, y) {
		b.px[y*b.w+x] = v
	}
}

// stamp sets every '#' of a, with its top-left at (x, y), to v. Dots that
// fall outside the bitmap are dropped.
func (b *bitmap) stamp(x, y int, a art, v bool) {
	for j, row := range a {
		for i := 0; i < len(row); i++ {
			if row[i] == '#' {
				b.set(x+i, y+j, v)
			}
		}
	}
}

// dotBit[row][col] is the Braille dot bit for that position in a cell.
var dotBit = [4][2]rune{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

// String renders the bitmap as Braille rows joined by "\n". A cell with no
// dots is a plain space, which every font draws one column wide.
func (b *bitmap) String() string {
	rows := make([]string, b.h/4)
	for cy := range rows {
		var sb strings.Builder
		for cx := 0; cx < b.w/2; cx++ {
			var bits rune
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					if b.get(cx*2+dx, cy*4+dy) {
						bits |= dotBit[dy][dx]
					}
				}
			}
			if bits == 0 {
				sb.WriteByte(' ')
			} else {
				sb.WriteRune(0x2800 + bits)
			}
		}
		rows[cy] = sb.String()
	}
	return strings.Join(rows, "\n")
}
