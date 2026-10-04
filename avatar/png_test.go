package avatar_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/ows4444/tui/avatar"
	"github.com/ows4444/tui/imageview"
)

func decode(t *testing.T, b []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("PNG does not decode: %v", err)
	}
	return img
}

// PNG is a square image of the size asked for that image/png decodes, and a
// size above MaxPNGSize is drawn at MaxPNGSize.
func TestPNGIsASquareImageOfTheSizeAsked(t *testing.T) {
	for _, size := range []int{1, 16, 64, 200} {
		img := decode(t, avatar.New("alain00").PNG(size))
		if b := img.Bounds(); b.Dx() != size || b.Dy() != size {
			t.Errorf("PNG(%d) is %dx%d", size, b.Dx(), b.Dy())
		}
	}
	if b := decode(t, avatar.New("alain00").PNG(avatar.MaxPNGSize+300)).Bounds(); b.Dx() != avatar.MaxPNGSize || b.Dy() != avatar.MaxPNGSize {
		t.Errorf("an oversize PNG is %dx%d, want %d square", b.Dx(), b.Dy(), avatar.MaxPNGSize)
	}
	for _, size := range []int{0, -5} {
		if got := avatar.New("alain00").PNG(size); got != nil {
			t.Errorf("PNG(%d) returned %d bytes", size, len(got))
		}
	}
}

// The same Model returns the same bytes, and another name other bytes.
func TestPNGIsDeterministic(t *testing.T) {
	for _, name := range names {
		a, b := avatar.New(name).PNG(48), avatar.New(name).PNG(48)
		if !bytes.Equal(a, b) {
			t.Errorf("%q: two renders differ", name)
		}
	}
	if bytes.Equal(avatar.New("ada").PNG(48), avatar.New("linus").PNG(48)) {
		t.Error("two names returned the same image")
	}
}

// The image is the avatar: the body colour in the middle of the body, the
// eye colour on an eye, nothing in the corners, and soft edges between.
func TestPNGDrawsTheAvatarsColours(t *testing.T) {
	m := avatar.New("alain00")
	body, eyes, plate := m.Colors()
	img := decode(t, m.PNG(128))
	count := map[color.NRGBA]int{}
	soft := 0
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A != 0 && c.A != 255 {
				soft++
				continue
			}
			count[c]++
		}
	}
	solid := func(c struct{ R, G, B uint8 }) color.NRGBA { return color.NRGBA{c.R, c.G, c.B, 255} }
	if count[solid(body)] < 128*128/3 {
		t.Errorf("only %d pixels are the body colour", count[solid(body)])
	}
	if n := count[solid(eyes)]; n < 50 || n > count[solid(body)]/4 {
		t.Errorf("%d pixels are the eye colour", n)
	}
	if count[color.NRGBA{}] < 128*128/10 {
		t.Errorf("only %d pixels are transparent", count[color.NRGBA{}])
	}
	if soft == 0 {
		t.Error("no pixel is partly transparent: the edge is not smooth")
	}
	if got := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA); got.A != 0 {
		t.Errorf("the corner is %v, want transparent", got)
	}

	// With a square plate the whole image is covered and the corner is the
	// plate's colour.
	m.Background = avatar.BackgroundSquare
	if got := color.NRGBAModel.Convert(decode(t, m.PNG(64)).At(0, 0)).(color.NRGBA); got != solid(plate) {
		t.Errorf("the corner over a square plate is %v, want %v", got, plate)
	}
}

// PNG draws the avatar as it is now, as View does.
func TestPNGFollowsThePose(t *testing.T) {
	rest := avatar.New("alain00")
	seen := map[string]bool{string(rest.PNG(64)): true}
	changes := map[string]func(*avatar.Model){
		"expression": func(m *avatar.Model) { m.Expression = avatar.ExpressionSurprised },
		"look":       func(m *avatar.Model) { m.LookAt(100, 0) },
		"blink":      func(m *avatar.Model) { m.Blink() },
		"hue":        func(m *avatar.Model) { m.Hue = 140 },
		"silhouette": func(m *avatar.Model) { m.Silhouette = avatar.SilhouetteSun },
	}
	for what, change := range changes {
		m := avatar.New("alain00")
		change(&m)
		if got := string(m.PNG(64)); seen[got] {
			t.Errorf("%s did not change the image", what)
		}
	}
}

// The bytes are what imageview draws on a terminal with kitty graphics.
func TestPNGGoesThroughImageview(t *testing.T) {
	m := avatar.New("alain00")
	img := imageview.New(m.PNG(64), m.Width, m.Height, m.Linearize())
	img.Kitty = true
	img.Getenv = func(string) string { return "" }
	view := img.View()
	if !strings.Contains(view, "\x1b_G") {
		t.Errorf("imageview did not transmit the image: %q", view[:min(len(view), 60)])
	}
	if rows := strings.Split(view, "\n"); len(rows) != m.Height {
		t.Errorf("imageview drew %d rows, want %d", len(rows), m.Height)
	}
	if img.Linearize() != "Avatar: alain00" {
		t.Errorf("the alt text is %q", img.Linearize())
	}
}
