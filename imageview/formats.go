package imageview

import (
	"bytes"
	"image"
	"image/draw"
	"image/gif"
	_ "image/jpeg" // registers the JPEG decoder for image.Decode
	"image/png"
	"strings"
	"time"

	"github.com/ows4444/tui/ansi"
)

// pngMagic is the first bytes of every PNG file.
var pngMagic = []byte("\x89PNG\r\n\x1a\n")

func isPNG(b []byte) bool { return bytes.HasPrefix(b, pngMagic) }

// decodeImage decodes a PNG, JPEG or GIF (its first frame).
func decodeImage(b []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(b))
	return img, err
}

// EncodePNG encodes img as a PNG, the form Model.PNG holds.
func EncodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ToPNG returns data as a PNG: unchanged when it already is one, otherwise
// decoded (JPEG, or the first frame of a GIF) and encoded again. It fails for
// data in a format that is not registered with the image package.
func ToPNG(data []byte) ([]byte, error) {
	if isPNG(data) {
		return data, nil
	}
	img, err := decodeImage(data)
	if err != nil {
		return nil, err
	}
	return EncodePNG(img)
}

// FromImage is New for an image.Image, such as one a program has drawn or
// decoded itself. It reports an error when the image cannot be encoded.
func FromImage(img image.Image, width, height int, alt string) (Model, error) {
	b, err := EncodePNG(img)
	if err != nil {
		return Model{}, err
	}
	return New(b, width, height, alt), nil
}

// Frame is one picture of an animation: the PNG to show, and how long to show it.
type Frame struct {
	PNG   []byte
	Delay time.Duration
}

// Frames decodes an animated GIF into frames, each composed over the ones before
// it as the format says, so every PNG is a whole picture. Show them one after
// another by assigning each frame's PNG to Model.PNG when its Delay has passed
// (tui.Tick or motion.Tween drive that). A GIF with one picture gives one frame.
// A zero delay becomes 100 ms, as browsers treat it.
func Frames(data []byte) ([]Frame, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	canvas := image.NewRGBA(image.Rect(0, 0, g.Config.Width, g.Config.Height))
	out := make([]Frame, 0, len(g.Image))
	for i, fr := range g.Image {
		draw.Draw(canvas, fr.Bounds(), fr, fr.Bounds().Min, draw.Over)
		b, err := EncodePNG(canvas)
		if err != nil {
			return nil, err
		}
		d := time.Duration(g.Delay[i]) * 10 * time.Millisecond
		if d <= 0 {
			d = 100 * time.Millisecond
		}
		out = append(out, Frame{PNG: b, Delay: d})
		if i < len(g.Disposal) && g.Disposal[i] == gif.DisposalBackground {
			draw.Draw(canvas, fr.Bounds(), image.Transparent, image.Point{}, draw.Src)
		}
	}
	return out, nil
}

// HalfBlocks draws the image over cols by rows cells with half-block
// characters, two pixels to a cell: the upper half block in the top pixel's
// colour over the bottom pixel's. It needs only 24-bit colour, no graphics
// protocol, and is what Model draws when the terminal has none. Pixels more
// than half transparent are left unpainted. PNG, JPEG and GIF are read; "" is
// returned for data that is none of them or a size that is not positive. Rows
// are joined by "\n".
func HalfBlocks(data []byte, cols, rows int) string {
	if cols <= 0 || rows <= 0 {
		return ""
	}
	img, err := decodeImage(data)
	if err != nil {
		return ""
	}
	return halfBlocks(img, cols, rows)
}

func halfBlocks(img image.Image, cols, rows int) string {
	pix := scale(img, cols, rows*2)
	lines := make([]string, rows)
	var b []byte
	for y := 0; y < rows; y++ {
		b = b[:0]
		for x := 0; x < cols; x++ {
			top, bot := pix[(2*y)*cols+x], pix[(2*y+1)*cols+x]
			b = append(b, cell(top, bot)...)
		}
		lines[y] = string(b)
	}
	return strings.Join(lines, "\n")
}

const (
	upperHalf = "▀"
	lowerHalf = "▄"
)

// cell is one character cell for a pair of pixels.
func cell(top, bot px) string {
	rgb := func(p px) ansi.RGB { return ansi.RGB{R: p.r, G: p.g, B: p.b} }
	switch {
	case top.on && bot.on:
		return ansi.NewStyle().Foreground(rgb(top)).Background(rgb(bot)).Render(upperHalf)
	case top.on:
		return ansi.NewStyle().Foreground(rgb(top)).Render(upperHalf)
	case bot.on:
		return ansi.NewStyle().Foreground(rgb(bot)).Render(lowerHalf)
	}
	return " "
}
