package imageview

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// halves is a 2x2 image: red over blue on the left, green over white on the right.
func halves() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(0, 1, color.RGBA{0, 0, 255, 255})
	img.Set(1, 0, color.RGBA{0, 255, 0, 255})
	img.Set(1, 1, color.RGBA{255, 255, 255, 255})
	return img
}

func encode(t *testing.T, enc func(*bytes.Buffer, image.Image) error, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := enc(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestHalfBlocksTwoPixelsToACell(t *testing.T) {
	data := encode(t, func(b *bytes.Buffer, i image.Image) error { return png.Encode(b, i) }, halves())
	got := HalfBlocks(data, 2, 1)
	cells := strings.Split(strings.ReplaceAll(got, "\x1b[0m", "\x1b[0m\n"), "\n")
	want := []string{
		ansi.NewStyle().Foreground(ansi.RGB{R: 255}).Background(ansi.RGB{B: 255}).Render("\u2580"),
		ansi.NewStyle().Foreground(ansi.RGB{G: 255}).Background(ansi.RGB{R: 255, G: 255, B: 255}).Render("\u2580"),
	}
	if cells[0] != want[0] || cells[1] != want[1] {
		t.Fatalf("got %q\nwant %q", got, strings.Join(want, ""))
	}
	if HalfBlocks(data, 0, 3) != "" || HalfBlocks([]byte("nope"), 2, 1) != "" {
		t.Fatal("a bad size or bad data should give nothing")
	}
}

func TestHalfBlocksTransparency(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	img.Set(0, 0, color.NRGBA{200, 0, 0, 255}) // top only
	img.Set(1, 1, color.NRGBA{0, 200, 0, 255}) // bottom only
	data := encode(t, func(b *bytes.Buffer, i image.Image) error { return png.Encode(b, i) }, img)
	got := HalfBlocks(data, 3, 1)
	if !strings.Contains(got, "\u2580") || !strings.Contains(got, "\u2584") || !strings.HasSuffix(got, " ") {
		t.Fatalf("expected an upper half, a lower half and a blank: %q", got)
	}
}

func TestFallbackIsBlocksUnlessPlaceholderOnlyOrASCII(t *testing.T) {
	data := encode(t, func(b *bytes.Buffer, i image.Image) error { return png.Encode(b, i) }, halves())
	m := New(data, 4, 2, "pic")
	v := m.View()
	if !strings.Contains(v, "\u2580") {
		t.Fatalf("no half blocks: %q", v)
	}
	lines := strings.Split(v, "\n")
	if len(lines) != 2 || ansi.Width(lines[0]) != 4 {
		t.Fatalf("size: %q", v)
	}
	m.PlaceholderOnly = true
	if strings.Contains(m.View(), "\u2580") {
		t.Fatal("PlaceholderOnly still drew blocks")
	}
	m = New(data, 8, 4, "pic")
	m.Theme = theme.DarkTheme()
	m.Theme.Glyphs = theme.ASCIIGlyphSet()
	if strings.Contains(m.View(), "\u2580") || !strings.Contains(ansi.StripANSI(m.View()), "pic") {
		t.Fatal("an ASCII glyph set must get the text placeholder")
	}
	bad := New([]byte("not an image"), 8, 4, "pic")
	if !strings.Contains(ansi.StripANSI(bad.View()), "pic") {
		t.Fatal("an undecodable image should fall back to the placeholder")
	}
}

func TestNewTakesJPEGAndGIF(t *testing.T) {
	jpg := encode(t, func(b *bytes.Buffer, i image.Image) error { return jpeg.Encode(b, i, nil) }, halves())
	gf := encode(t, func(b *bytes.Buffer, i image.Image) error { return gif.Encode(b, i, nil) }, halves())
	for name, data := range map[string][]byte{"jpeg": jpg, "gif": gf} {
		m := New(data, 2, 1, name)
		if !isPNG(m.PNG) {
			t.Errorf("%s was not converted to PNG", name)
		}
		if !strings.Contains(m.View(), "\u2580") {
			t.Errorf("%s did not draw", name)
		}
	}
	if m := New([]byte("garbage"), 2, 1, "x"); string(m.PNG) != "garbage" {
		t.Error("undecodable data should be left as given")
	}
	got, err := ToPNG(jpg)
	if err != nil || !isPNG(got) {
		t.Fatalf("ToPNG: %v", err)
	}
	same, _ := ToPNG(got)
	if &same[0] != &got[0] {
		t.Error("ToPNG should return a PNG unchanged")
	}
	if _, err := ToPNG([]byte("x")); err == nil {
		t.Error("ToPNG accepted garbage")
	}
}

func TestFromImage(t *testing.T) {
	m, err := FromImage(halves(), 2, 1, "x")
	if err != nil || !isPNG(m.PNG) || m.Width != 2 {
		t.Fatalf("%v %+v", err, m)
	}
}

func TestFramesComposeAndKeepDelays(t *testing.T) {
	pal := color.Palette{color.RGBA{0, 0, 0, 0}, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}}
	first := image.NewPaletted(image.Rect(0, 0, 2, 1), pal)
	first.SetColorIndex(0, 0, 1)
	first.SetColorIndex(1, 0, 1)
	second := image.NewPaletted(image.Rect(1, 0, 2, 1), pal) // changes only the right pixel
	second.SetColorIndex(1, 0, 2)
	g := &gif.GIF{Image: []*image.Paletted{first, second}, Delay: []int{5, 0}, Disposal: []byte{gif.DisposalNone, gif.DisposalNone}, Config: image.Config{Width: 2, Height: 1}}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		t.Fatal(err)
	}
	frames, err := Frames(buf.Bytes())
	if err != nil || len(frames) != 2 {
		t.Fatalf("%v, %d frames", err, len(frames))
	}
	if frames[0].Delay != 50*time.Millisecond || frames[1].Delay != 100*time.Millisecond {
		t.Fatalf("delays %v %v", frames[0].Delay, frames[1].Delay)
	}
	img, _ := decodeImage(frames[1].PNG)
	l, r := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA), color.NRGBAModel.Convert(img.At(1, 0)).(color.NRGBA)
	if l.R != 255 || r.B != 255 {
		t.Fatalf("frame 2 is not composed over frame 1: left %v right %v", l, r)
	}
	if _, err := Frames([]byte("nope")); err == nil {
		t.Fatal("Frames accepted garbage")
	}
}
