package imageview

import (
	"bytes"
	"go/parser"
	"go/token"
	"image"
	"image/color"
	"image/png"
	"strconv"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/render"
)

// twoColourPNG is w by h pixels: the left half red, the right half blue, with
// a fully transparent top-left pixel.
func twoColourPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBA{255, 0, 0, 255}
			if x >= w/2 {
				c = color.NRGBA{0, 0, 255, 255}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	img.SetNRGBA(0, 0, color.NRGBA{})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// sixelImage is what a terminal would draw for a Sixel sequence.
type sixelImage struct {
	w, h    int
	palette map[int][3]int // percent RGB
	pix     map[[2]int]int // (x, y) to colour register
}

// decodeSixel is a small Sixel reader, enough for what Sixel writes.
func decodeSixel(t *testing.T, s string) sixelImage {
	t.Helper()
	body, ok := strings.CutPrefix(s, "\x1bPq")
	if !ok || !strings.HasSuffix(body, "\x1b\\") {
		t.Fatalf("not a Sixel DCS: %.20q", s)
	}
	body = strings.TrimSuffix(body, "\x1b\\")
	im := sixelImage{palette: map[int][3]int{}, pix: map[[2]int]int{}}
	num := func(i *int) int {
		n := 0
		for *i < len(body) && body[*i] >= '0' && body[*i] <= '9' {
			n = n*10 + int(body[*i]-'0')
			*i++
		}
		return n
	}
	x, band, cur := 0, 0, 0
	for i := 0; i < len(body); {
		switch c := body[i]; {
		case c == '"':
			i++
			num(&i) // pan
			i++
			num(&i) // pad
			i++
			im.w = num(&i)
			i++
			im.h = num(&i)
		case c == '#':
			i++
			cur = num(&i)
			if i < len(body) && body[i] == ';' {
				i++
				num(&i) // colour space, 2 = RGB
				var rgb [3]int
				for k := range rgb {
					i++
					rgb[k] = num(&i)
				}
				im.palette[cur] = rgb
			}
		case c == '$':
			x = 0
			i++
		case c == '-':
			x, band = 0, band+1
			i++
		case c == '!':
			i++
			n := num(&i)
			bits := body[i] - '?'
			i++
			for k := 0; k < n; k++ {
				im.set(x, band, bits, cur)
				x++
			}
		case c >= '?' && c <= '~':
			im.set(x, band, c-'?', cur)
			x++
			i++
		default:
			t.Fatalf("unexpected byte %q at %d", c, i)
		}
	}
	return im
}

func (im *sixelImage) set(x, band int, bits byte, reg int) {
	for dy := 0; dy < 6; dy++ {
		if bits&(1<<dy) != 0 {
			im.pix[[2]int{x, band*6 + dy}] = reg
		}
	}
}

func TestSixelRoundTrip(t *testing.T) {
	// 4 by 2 cells of 4 by 6 pixels: a 16 by 12 image, two sixel bands.
	got := Sixel(twoColourPNG(t, 8, 4), 4, 2, 4, 6)
	im := decodeSixel(t, got)
	if im.w != 16 || im.h != 12 {
		t.Fatalf("raster size %dx%d, want 16x12", im.w, im.h)
	}
	if len(im.palette) != 2 {
		t.Fatalf("palette has %d colours, want 2: %v", len(im.palette), im.palette)
	}
	red, blue := -1, -1
	for reg, c := range im.palette {
		switch c {
		case [3]int{100, 0, 0}:
			red = reg
		case [3]int{0, 0, 100}:
			blue = reg
		}
	}
	if red < 0 || blue < 0 {
		t.Fatalf("palette %v lacks pure red and blue", im.palette)
	}
	for y := 0; y < 12; y++ {
		for x := 0; x < 16; x++ {
			reg, drawn := im.pix[[2]int{x, y}]
			want := red
			if x >= 8 {
				want = blue
			}
			// The transparent source pixel scales to the 2 by 3 block at the corner.
			if x < 2 && y < 3 {
				if drawn {
					t.Errorf("(%d,%d) is transparent but drawn", x, y)
				}
				continue
			}
			if !drawn || reg != want {
				t.Errorf("(%d,%d) = register %d (drawn %v), want %d", x, y, reg, drawn, want)
			}
		}
	}
}

func TestSixelUsesDefaultCellSizeAndRejectsBadInput(t *testing.T) {
	im := decodeSixel(t, Sixel(twoColourPNG(t, 4, 4), 2, 1, 0, 0))
	if im.w != 2*DefaultCellWidth || im.h != DefaultCellHeight {
		t.Errorf("default cell size gave %dx%d", im.w, im.h)
	}
	for name, got := range map[string]string{
		"not a png": Sixel([]byte("nope"), 2, 2, 8, 16),
		"no cols":   Sixel(twoColourPNG(t, 4, 4), 0, 2, 8, 16),
		"no rows":   Sixel(twoColourPNG(t, 4, 4), 2, -1, 8, 16),
	} {
		if got != "" {
			t.Errorf("%s: got %.20q, want empty", name, got)
		}
	}
}

func TestSixelLimitsThePaletteTo256(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.SetNRGBA(x, y, color.NRGBA{uint8(x * 4), uint8(y * 4), uint8((x + y) * 2), 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	im := decodeSixel(t, Sixel(b.Bytes(), 8, 4, 8, 16))
	if n := len(im.palette); n == 0 || n > 256 {
		t.Errorf("palette has %d colours, want 1 to 256", n)
	}
	if got, again := Sixel(b.Bytes(), 8, 4, 8, 16), Sixel(b.Bytes(), 8, 4, 8, 16); got != again {
		t.Error("the encoding is not deterministic")
	}
}

func TestSixelRunLengthEncodesLongRuns(t *testing.T) {
	got := Sixel(twoColourPNG(t, 8, 4), 4, 2, 4, 6)
	if !strings.Contains(got, "!") {
		t.Errorf("a 16-pixel run was not run-length encoded: %q", got)
	}
	if n, err := strconv.Atoi(got[strings.Index(got, "!")+1 : strings.Index(got, "!")+2]); err != nil || n < 4 {
		t.Errorf("bad repeat count in %q", got)
	}
}

// Criterion: the encoder uses no package outside the standard library.
func TestSixelEncoderImportsOnlyTheStandardLibrary(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "sixel.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, im := range f.Imports {
		path, _ := strconv.Unquote(im.Path.Value)
		if first, _, _ := strings.Cut(path, "/"); strings.Contains(first, ".") {
			t.Errorf("sixel.go imports %s, which is outside the standard library", path)
		}
	}
}

func sixelModel(t *testing.T) Model {
	t.Helper()
	m := New(twoColourPNG(t, 8, 4), 6, 3, "alt")
	m.CellWidth, m.CellHeight = 4, 6
	return m
}

// Criterion #63: Sixel and not kitty gives a Sixel image.
func TestViewEmitsSixelWhenOnlySixelIsReported(t *testing.T) {
	m := sixelModel(t)
	m.Sixel = true
	rows := strings.Split(m.View(), "\n")
	if len(rows) != 3 || !strings.HasPrefix(rows[0], "\x1bPq") || strings.Contains(m.View(), "\x1b_G") {
		t.Fatalf("not a Sixel view: %.40q", m.View())
	}
	im := decodeSixel(t, strings.TrimSuffix(rows[0], strings.Repeat(" ", 6)))
	if im.w != 24 || im.h != 18 {
		t.Errorf("Sixel raster %dx%d, want 24x18 (6x3 cells of 4x6)", im.w, im.h)
	}
}

// Criterion #64: neither reported gives the text fallback.
func TestViewFallsBackToTextWhenNeitherIsReported(t *testing.T) {
	m := sixelModel(t)
	got := m.View()
	if strings.Contains(got, "\x1bP") || strings.Contains(got, "\x1b_") || !strings.Contains(ansi.StripANSI(got), "|alt") {
		t.Errorf("want the text placeholder, got %q", got)
	}
	m.Sixel = true
	m.PNG = []byte("not a png") // Sixel cannot draw it: placeholder, not a broken sequence
	if got := m.View(); strings.Contains(got, "\x1bP") {
		t.Errorf("an undecodable PNG produced a Sixel sequence: %.30q", got)
	}
}

func TestKittyStaysPreferredWhenBothAreReported(t *testing.T) {
	m := sixelModel(t)
	m.Kitty, m.Sixel = true, true
	got := m.View()
	if !strings.Contains(got, "\x1b_G") || strings.Contains(got, "\x1bPq") {
		t.Errorf("kitty must win over Sixel: %.40q", got)
	}
}

// The cell renderer draws a Sixel view without falling back, saves and restores
// the cursor around the image, and does not re-send it while its rows are
// unchanged.
func TestSixelViewThroughCellRenderer(t *testing.T) {
	m := sixelModel(t)
	m.Sixel = true
	c := render.New()
	prev := 0
	draw := func(lines ...string) string {
		t.Helper()
		out, st, ok := c.Frame(render.Frame{
			Lines: lines, Max: max(len(lines), prev), PrevRows: prev, Width: 40, Height: 100,
			RegionTop: func() string { return "" }, FitLine: func(s string) string { return s },
		})
		if !ok || c.Reason() != "" || len(st.Fallbacks) != 0 {
			t.Fatalf("fell back: ok=%v reason=%q rows=%v", ok, c.Reason(), st.Fallbacks)
		}
		prev = max(len(lines), prev)
		return out
	}
	img := strings.Split(m.View(), "\n")
	out := draw(append([]string{"title"}, img...)...)
	i := strings.Index(out, "\x1bPq")
	if i < 2 || out[i-2:i] != "\x1b7" || !strings.Contains(out[i:], "\x1b\\\x1b8") {
		t.Fatalf("Sixel not wrapped in cursor save/restore: %.80q", out)
	}
	if out := draw(append([]string{"TITLE"}, img...)...); strings.Contains(out, "\x1bPq") {
		t.Errorf("unchanged Sixel rows re-sent: %.60q", out)
	}
}
