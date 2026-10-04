package avatar

import (
	"bytes"
	"image"
	"image/png"
	"math"

	"github.com/ows4444/tui/ansi"
)

// MaxPNGSize is the largest side, in pixels, PNG draws; a larger size is
// drawn at this one.
const MaxPNGSize = 512

// PNG returns the avatar as a square PNG image, size pixels on a side, with
// smooth edges and a transparent background where there is no plate. Like
// View it draws the avatar as it is now: its Expression, where it looks and
// a blink in progress. It is the picture for a terminal that shows images:
//
//	img := imageview.New(m.PNG(128), m.Width, m.Height, m.Linearize())
//	img.Kitty = caps.KittyGraphics
//
// with View as the fallback elsewhere. The same Model always returns the same
// bytes. A Model from New keeps its recent images and returns the same slice
// for the same picture, so imageview, which encodes a slice once, redraws
// only when the avatar does: do not write to the slice. A size of zero or
// less returns nil.
func (m Model) PNG(size int) []byte {
	if size <= 0 {
		return nil
	}
	size = min(size, MaxPNGSize)
	c := m.cache
	if c == nil {
		return m.encode(size)
	}
	key := m.key(size, size)
	c.mu.Lock()
	defer c.mu.Unlock()
	img, ok := c.pngs.get(key)
	if !ok {
		img = m.encode(size)
		c.pngs.put(key, img)
	}
	return img
}

// encode draws the avatar size pixels on a side and encodes it.
func (m Model) encode(size int) []byte {
	if drawHook != nil {
		drawHook()
	}
	t := m.traits()
	lookX, zoom := m.moved()
	f := layoutFigure(t)
	s := newScene(f, m.plate())
	from, to, at := m.poses()
	s.setEyes(f.posed(from, to, at, unit(unit(m.LookX)+lookX), unit(m.LookY), m.tremble(to, f, 0), blinkOpen[m.blink%len(blinkOpen)], 1))
	s.zoom = zoom
	s.setLift(lift(from, to, at), 0)
	p := m.palette(t)
	colour := [layers]ansi.RGB{layerPlate: p.bg, layerBody: p.head, layerEye: p.eye}

	cx, cy, side := s.bounds()
	unit := side / float64(size)
	x0, y0 := cx-side/2, cy-side/2
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	// The layer at every corner of the pixel lattice, sampled once: a pixel
	// whose four corners and centre agree lies wholly inside one layer and
	// needs no more samples. That is nearly all of them; only the pixels an
	// outline crosses are sampled finely.
	corner := make([]layer, (size+1)*(size+1))
	for j := 0; j <= size; j++ {
		for i := 0; i <= size; i++ {
			corner[j*(size+1)+i] = s.at(x0+float64(i)*unit, y0+float64(j)*unit)
		}
	}
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			o := img.PixOffset(i, j)
			at := j*(size+1) + i
			if l := corner[at]; l == corner[at+1] && l == corner[at+size+1] && l == corner[at+size+2] &&
				l == s.at(x0+(float64(i)+0.5)*unit, y0+(float64(j)+0.5)*unit) {
				if l != layerNone {
					c := colour[l]
					img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = c.R, c.G, c.B, 255
				}
				continue
			}
			// The pixel is the mean of the layers under its samples; its
			// alpha is the share of them that hit the figure.
			var r, g, b, hit float64
			for sb := 0; sb < samples; sb++ {
				for sa := 0; sa < samples; sa++ {
					x := x0 + (float64(i)+(float64(sa)+0.5)/samples)*unit
					y := y0 + (float64(j)+(float64(sb)+0.5)/samples)*unit
					l := s.at(x, y)
					if l == layerNone {
						continue
					}
					c := colour[l]
					r, g, b, hit = r+float64(c.R), g+float64(c.G), b+float64(c.B), hit+1
				}
			}
			if hit == 0 {
				continue
			}
			img.Pix[o], img.Pix[o+1], img.Pix[o+2] = shade(r/hit), shade(g/hit), shade(b/hit)
			img.Pix[o+3] = shade(255 * hit / (samples * samples))
		}
	}
	var buf bytes.Buffer
	// Encoding to a bytes.Buffer cannot fail: the image is non-empty and the
	// writer never returns an error. The image is redrawn as the avatar
	// moves, so encoding favours speed over size.
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	_ = enc.Encode(&buf, img)
	return buf.Bytes()
}

// shade rounds a channel value in [0, 255] to a byte.
func shade(v float64) uint8 {
	return uint8(math.Round(v)) // #nosec G115 -- v is a mean of bytes, or 255 times a share
}
