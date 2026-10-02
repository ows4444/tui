package imageview

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"sort"
	"strconv"
	"strings"
)

// DefaultCellWidth and DefaultCellHeight are the pixel size of a terminal cell
// that Sixel assumes when Model.CellWidth and Model.CellHeight are zero: the
// size of an 80 by 24 cell grid on a typical 8 by 16 pixel font.
const (
	DefaultCellWidth  = 8
	DefaultCellHeight = 16
)

// maxSixelColors is the palette size. Sixel terminals guarantee at least 256
// registers; the encoder uses at most this many.
const maxSixelColors = 256

// Sixel returns the DEC Sixel sequence that draws pngData scaled to cols by rows
// cells of cellW by cellH pixels each (zero means DefaultCellWidth and
// DefaultCellHeight), or "" when pngData does not decode or the size is not
// positive. The image is quantised to at most 256 colours with a median cut;
// pixels with alpha below one half are left undrawn. The sequence draws from
// the cursor and moves it, so the renderer saves and restores the cursor around
// it (see the zero-width rule in package render).
func Sixel(pngData []byte, cols, rows, cellW, cellH int) string {
	if cellW <= 0 {
		cellW = DefaultCellWidth
	}
	if cellH <= 0 {
		cellH = DefaultCellHeight
	}
	if cols <= 0 || rows <= 0 {
		return ""
	}
	img, err := decodePNG(pngData)
	if err != nil {
		return ""
	}
	return encodeSixel(img, cols*cellW, rows*cellH)
}

func decodePNG(b []byte) (image.Image, error) { return png.Decode(bytes.NewReader(b)) }

// px is one scaled pixel: its colour in 8 bits per channel, and whether it is
// drawn at all.
type px struct {
	r, g, b uint8
	on      bool
}

// scale resamples img to w by h pixels, taking the nearest source pixel.
func scale(img image.Image, w, h int) []px {
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	out := make([]px, w*h)
	if sw == 0 || sh == 0 {
		return out
	}
	for y := 0; y < h; y++ {
		sy := b.Min.Y + y*sh/h
		for x := 0; x < w; x++ {
			sx := b.Min.X + x*sw/w
			r, g, bl, a := img.At(sx, sy).RGBA()
			if a < 0x8000 {
				continue
			}
			// RGBA is alpha-premultiplied; undo it so a half-transparent red is red.
			out[y*w+x] = px{uint8(r * 0xff / a), uint8(g * 0xff / a), uint8(bl * 0xff / a), true} // #nosec G115 -- r, g, bl <= a, so each quotient is at most 255
		}
	}
	return out
}

// box is a set of pixel colours the median cut may still split.
type box struct{ cols []color.RGBA }

func (b box) span() (ch int, width int) {
	var lo, hi [3]int
	for i := range lo {
		lo[i], hi[i] = 255, 0
	}
	for _, c := range b.cols {
		for i, v := range [3]uint8{c.R, c.G, c.B} {
			lo[i], hi[i] = min(lo[i], int(v)), max(hi[i], int(v))
		}
	}
	for i := range lo {
		if hi[i]-lo[i] >= width {
			ch, width = i, hi[i]-lo[i]
		}
	}
	return ch, width
}

func channel(c color.RGBA, ch int) uint8 { return [3]uint8{c.R, c.G, c.B}[ch] }

// medianCut returns at most n representative colours of cols.
func medianCut(cols []color.RGBA, n int) []color.RGBA {
	boxes := []box{{cols}}
	for len(boxes) < n {
		// Split the box with the widest channel range.
		best, bestW := -1, 0
		for i, b := range boxes {
			if len(b.cols) < 2 {
				continue
			}
			if _, w := b.span(); w > bestW {
				best, bestW = i, w
			}
		}
		if best < 0 {
			break
		}
		b := boxes[best]
		ch, _ := b.span()
		sort.Slice(b.cols, func(i, j int) bool { return channel(b.cols[i], ch) < channel(b.cols[j], ch) })
		mid := len(b.cols) / 2
		boxes[best] = box{b.cols[:mid]}
		boxes = append(boxes, box{b.cols[mid:]})
	}
	pal := make([]color.RGBA, 0, len(boxes))
	for _, b := range boxes {
		var r, g, bl int
		for _, c := range b.cols {
			r, g, bl = r+int(c.R), g+int(c.G), bl+int(c.B)
		}
		n := len(b.cols)
		if n == 0 {
			continue
		}
		pal = append(pal, color.RGBA{uint8(r / n), uint8(g / n), uint8(bl / n), 255}) // #nosec G115 -- the mean of values that are at most 255
	}
	return pal
}

// encodeSixel quantises the scaled image and writes the Sixel sequence.
func encodeSixel(img image.Image, w, h int) string {
	pix := scale(img, w, h)
	// Collect the distinct colours (5 bits per channel keeps the set small and
	// the median cut fast), weighted by occurrence.
	count := map[uint16]int{}
	for _, p := range pix {
		if p.on {
			count[key(p)]++
		}
	}
	cols := make([]color.RGBA, 0, len(count))
	for k := range count {
		r, g, b := unkey(k)
		cols = append(cols, color.RGBA{r, g, b, 255})
	}
	sort.Slice(cols, func(i, j int) bool { // map order is random: keep the output stable
		a, b := cols[i], cols[j]
		return uint32(a.R)<<16|uint32(a.G)<<8|uint32(a.B) < uint32(b.R)<<16|uint32(b.G)<<8|uint32(b.B)
	})
	pal := medianCut(cols, maxSixelColors)
	nearest := map[uint16]int{}
	index := func(p px) int {
		k := key(p)
		if i, ok := nearest[k]; ok {
			return i
		}
		best, bd := 0, 1<<30
		for i, c := range pal {
			dr, dg, db := int(c.R)-int(p.r), int(c.G)-int(p.g), int(c.B)-int(p.b)
			if d := dr*dr + dg*dg + db*db; d < bd {
				best, bd = i, d
			}
		}
		nearest[k] = best
		return best
	}

	var sb strings.Builder
	sb.WriteString("\x1bPq\"1;1;" + strconv.Itoa(w) + ";" + strconv.Itoa(h))
	for i, c := range pal {
		sb.WriteString("#" + strconv.Itoa(i) + ";2;" + strconv.Itoa(int(c.R)*100/255) + ";" + strconv.Itoa(int(c.G)*100/255) + ";" + strconv.Itoa(int(c.B)*100/255))
	}
	idx := make([]int, len(pix)) // palette index per pixel, -1 undrawn
	for i, p := range pix {
		if p.on {
			idx[i] = index(p)
		} else {
			idx[i] = -1
		}
	}
	for y0 := 0; y0 < h; y0 += 6 {
		// The colours that appear in this band of six rows, in palette order.
		used := make([]bool, len(pal))
		for y := y0; y < min(y0+6, h); y++ {
			for x := 0; x < w; x++ {
				if i := idx[y*w+x]; i >= 0 {
					used[i] = true
				}
			}
		}
		first := true
		for ci, ok := range used {
			if !ok {
				continue
			}
			if !first {
				sb.WriteByte('$') // back to the band's left edge for the next colour
			}
			first = false
			sb.WriteString("#" + strconv.Itoa(ci))
			var run byte
			n := 0
			flush := func() {
				switch {
				case n == 0:
				case n > 3:
					sb.WriteString("!" + strconv.Itoa(n))
					sb.WriteByte(run)
				default:
					for ; n > 0; n-- {
						sb.WriteByte(run)
					}
				}
				n = 0
			}
			for x := 0; x < w; x++ {
				var bits byte
				for dy := 0; dy < 6 && y0+dy < h; dy++ {
					if idx[(y0+dy)*w+x] == ci {
						bits |= 1 << dy
					}
				}
				c := '?' + bits
				if n > 0 && c != run {
					flush()
				}
				run = c
				n++
			}
			flush()
		}
		sb.WriteByte('-') // next band
	}
	sb.WriteString("\x1b\\")
	return sb.String()
}

func key(p px) uint16 { return uint16(p.r>>3)<<10 | uint16(p.g>>3)<<5 | uint16(p.b>>3) }

func unkey(k uint16) (r, g, b uint8) {
	// Expand 5 bits back to 8, replicating the high bits so 31 maps to 255.
	e := func(v uint16) uint8 { v &= 31; return uint8(v<<3 | v>>2) } // #nosec G115 -- v is at most 31, so the result is at most 255
	return e(k >> 10), e(k >> 5), e(k)
}
